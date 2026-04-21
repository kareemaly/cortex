package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/kareemaly/cortex/internal/session"
)

// Publisher is the sink for status transitions. The production path posts to
// cortexd's /agent/status endpoint; tests substitute an in-memory recorder.
type Publisher func(Transition)

// Transition is a single status change forwarded from the Hub. The supervisor
// calls the publisher for every Hub event — deduplication is left to the
// server side.
type Transition struct {
	SessionID string
	TicketID  string
	Status    session.AgentStatus
	Tool      *string
	Work      *string
	At        time.Time
}

// SupervisorConfig carries everything a supervisor needs to run for one
// session: identity, the Hub event source, file paths for liveness, and the
// publisher for state transitions.
type SupervisorConfig struct {
	SessionID     string
	TicketID      string
	ArchitectPath string
	LivenessPath  string

	// HubEventSource, when non-nil, is called once during StartSupervisor to
	// obtain a channel of Hub-sourced status events bound to the given context.
	// When nil, no Hub forwarding occurs (liveness-only supervision).
	HubEventSource func(ctx context.Context) <-chan HubEvent

	// EndFunc is called when the liveness loop detects the session has ended.
	// Defaults to calling DELETE {DaemonURL}/sessions/{SessionID} when nil.
	EndFunc func()

	// Publisher reports status transitions. Defaults to HTTPPublisher against
	// DaemonURL when nil.
	Publisher Publisher
	DaemonURL string

	// Logger is used for warnings. Defaults to slog.Default() when nil.
	Logger *slog.Logger

	// LivenessInterval controls how often the liveness file is stat-ed.
	// Zero picks defaultLivenessInterval.
	LivenessInterval time.Duration

	// now returns the current time. Tests override it for determinism.
	now func() time.Time
}

const defaultLivenessInterval = 1 * time.Second

// StartSupervisor wires a supervisor for one session and starts its
// goroutines. The returned cancel func tears them down.
//
// Required: SessionID-or-TicketID and LivenessPath.
func StartSupervisor(ctx context.Context, cfg SupervisorConfig) (context.CancelFunc, error) {
	if cfg.SessionID == "" && cfg.TicketID == "" {
		return nil, errMissing("SessionID or TicketID")
	}
	if cfg.LivenessPath == "" {
		return nil, errMissing("LivenessPath")
	}
	if cfg.LivenessInterval == 0 {
		cfg.LivenessInterval = defaultLivenessInterval
	}
	if cfg.now == nil {
		cfg.now = time.Now
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Publisher == nil {
		cfg.Publisher = HTTPPublisherWithLogger(cfg.DaemonURL, cfg.ArchitectPath, cfg.Logger)
	}

	ctx, cancelCtx := context.WithCancel(ctx)

	var hubEvents <-chan HubEvent
	if cfg.HubEventSource != nil {
		hubEvents = cfg.HubEventSource(ctx)
	}

	sup := &supervisor{
		cfg:       cfg,
		ctx:       ctx,
		hubEvents: hubEvents,
	}

	if hubEvents != nil {
		sup.wg.Add(1)
		go sup.hubLoop()
	}
	sup.wg.Add(1)
	go sup.livenessLoop()

	return func() {
		cancelCtx()
		sup.wg.Wait()
	}, nil
}

type missingFieldError string

func (e missingFieldError) Error() string { return "agent: supervisor config missing " + string(e) }

func errMissing(field string) error { return missingFieldError(field) }

// supervisor holds the runtime state for one session.
type supervisor struct {
	cfg       SupervisorConfig
	ctx       context.Context
	hubEvents <-chan HubEvent
	wg        sync.WaitGroup
}

// hubLoop reads Hub events and forwards each one to the Publisher.
func (s *supervisor) hubLoop() {
	defer s.wg.Done()
	for {
		select {
		case <-s.ctx.Done():
			return
		case ev, ok := <-s.hubEvents:
			if !ok {
				return
			}
			trans := Transition{
				SessionID: s.cfg.SessionID,
				TicketID:  s.cfg.TicketID,
				Status:    ev.Status,
				At:        s.cfg.now(),
			}
			if ev.Tool != "" {
				t := ev.Tool
				trans.Tool = &t
			}
			if ev.Work != "" {
				w := ev.Work
				trans.Work = &w
			}
			s.cfg.Publisher(trans)
		}
	}
}

// livenessLoop polls the liveness file. When it disappears the session has
// ended: call EndFunc (or DELETE /sessions/{id} by default) and exit.
func (s *supervisor) livenessLoop() {
	defer s.wg.Done()
	for {
		if !sleep(s.ctx, s.cfg.LivenessInterval) {
			return
		}
		if _, err := os.Stat(s.cfg.LivenessPath); os.IsNotExist(err) {
			s.endSession()
			return
		}
	}
}

func (s *supervisor) endSession() {
	if s.cfg.EndFunc != nil {
		s.cfg.EndFunc()
		return
	}
	if s.cfg.DaemonURL == "" || s.cfg.SessionID == "" {
		return
	}
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodDelete,
		s.cfg.DaemonURL+"/sessions/"+s.cfg.SessionID, nil)
	if err != nil {
		s.cfg.Logger.Warn("supervisor: end session request failed",
			"error", err, "session_id", s.cfg.SessionID)
		return
	}
	if s.cfg.ArchitectPath != "" {
		req.Header.Set("X-Cortex-Architect", s.cfg.ArchitectPath)
	}
	resp, err := client.Do(req)
	if err != nil {
		s.cfg.Logger.Warn("supervisor: end session DELETE failed",
			"error", err, "session_id", s.cfg.SessionID)
		return
	}
	_ = resp.Body.Close()
	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusNotFound {
		s.cfg.Logger.Warn("supervisor: end session returned error",
			"status_code", resp.StatusCode, "session_id", s.cfg.SessionID)
	}
}

// sleep blocks for d or until ctx is done. Returns false if the context was
// cancelled.
func sleep(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		select {
		case <-ctx.Done():
			return false
		default:
			return true
		}
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// HTTPPublisher returns a Publisher that posts transitions to cortexd's
// /agent/status endpoint. daemonURL and architectPath are captured by closure.
// Failures are logged but not retried.
func HTTPPublisher(daemonURL, architectPath string) Publisher {
	return HTTPPublisherWithLogger(daemonURL, architectPath, nil)
}

// HTTPPublisherWithLogger is HTTPPublisher with an injectable logger.
func HTTPPublisherWithLogger(daemonURL, architectPath string, logger *slog.Logger) Publisher {
	if logger == nil {
		logger = slog.Default()
	}
	client := &http.Client{Timeout: 5 * time.Second}
	return func(t Transition) {
		payload := map[string]any{
			"status": string(t.Status),
		}
		if t.SessionID != "" {
			payload["session_id"] = t.SessionID
		}
		if t.Tool != nil {
			payload["tool"] = *t.Tool
		}
		if t.Work != nil {
			payload["work"] = *t.Work
		}
		body, err := json.Marshal(payload)
		if err != nil {
			logger.Warn("agent publisher: marshal failed",
				"error", err, "session_id", t.SessionID, "status", string(t.Status))
			return
		}
		req, err := http.NewRequest(http.MethodPost, daemonURL+"/agent/status", bytes.NewReader(body))
		if err != nil {
			logger.Warn("agent publisher: NewRequest failed",
				"error", err, "session_id", t.SessionID)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Cortex-Architect", architectPath)
		resp, err := client.Do(req)
		if err != nil {
			logger.Warn("agent publisher: POST failed",
				"error", err, "session_id", t.SessionID, "status", string(t.Status))
			return
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 400 {
			logger.Warn("agent publisher: non-2xx response",
				"status_code", resp.StatusCode, "session_id", t.SessionID, "status", string(t.Status))
		}
	}
}
