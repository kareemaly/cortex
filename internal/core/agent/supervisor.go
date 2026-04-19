package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/kareemaly/cortex/internal/session"
	"github.com/kareemaly/cortex/internal/tmux/observer"
)

// Publisher is the sink for status transitions. The production path posts to
// cortexd's /agent/status endpoint; tests substitute an in-memory recorder.
type Publisher func(Transition)

// Transition is a single status change observed by the decision machine. Only
// transitions are emitted — identical-to-previous status updates are
// collapsed by the supervisor before the publisher is called.
type Transition struct {
	SessionID string
	TicketID  string
	Status    session.AgentStatus
	Tool      *string
	Work      *string
	At        time.Time
}

// SupervisorConfig carries everything a supervisor needs to run for one
// session: identity, the adapter (parser + phrases + window), the observer
// to subscribe to, file paths for discovery & liveness, and the publisher
// for state transitions.
type SupervisorConfig struct {
	SessionID     string
	TicketID      string
	ArchitectPath string
	WorkingDir    string
	LivenessPath  string

	Adapter  *Adapter
	Observer *observer.Observer

	// PaneTarget is the tmux "session:window.pane" string to register with
	// the observer. Empty disables pane observation for this session — OK
	// for agents whose plugin gives authoritative status (opencode), but
	// means Claude/Codex lose their awaiting_input signal.
	PaneTarget string

	// Runtime carries adapter-specific context for ResolveTranscript (env,
	// transcript hint from Prepare).
	Runtime RuntimeCtx

	// Publisher reports status transitions. Defaults to HTTPPublisher against
	// DaemonURL when nil.
	Publisher Publisher
	DaemonURL string

	// Logger is used for noisy warnings (discovery timeout, unexpected
	// errors). Defaults to slog.Default() when nil.
	Logger *slog.Logger

	// Timing knobs — zero values pick sensible defaults.
	DiscoveryInterval time.Duration
	FollowInterval    time.Duration
	LivenessInterval  time.Duration

	// now returns the current time. Tests override it for determinism.
	now func() time.Time
}

// Defaults used when the caller leaves timing knobs zero.
const (
	defaultDiscoveryInterval   = 250 * time.Millisecond
	defaultFollowInterval      = 100 * time.Millisecond
	defaultLivenessInterval    = 1 * time.Second
	transcriptLineBufferMaxLen = 1 << 20 // 1 MiB; oversized lines trigger loud fallback
)

// StartSupervisor wires a supervisor for one session and starts its
// goroutines. The returned cancel func tears them down.
//
// Required: Adapter, SessionID-or-TicketID, and LivenessPath. If the
// Adapter has AwaitingInputPhrases configured and Observer+PaneTarget are
// set, pane observation is registered; otherwise the supervisor runs on
// transcript + liveness alone.
func StartSupervisor(ctx context.Context, cfg SupervisorConfig) (context.CancelFunc, error) {
	if cfg.Adapter == nil {
		return nil, errMissing("Adapter")
	}
	if cfg.SessionID == "" && cfg.TicketID == "" {
		return nil, errMissing("SessionID or TicketID")
	}
	if cfg.LivenessPath == "" {
		return nil, errMissing("LivenessPath")
	}
	if cfg.DiscoveryInterval == 0 {
		cfg.DiscoveryInterval = defaultDiscoveryInterval
	}
	if cfg.FollowInterval == 0 {
		cfg.FollowInterval = defaultFollowInterval
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
	decision := NewDecision(DecisionConfig{
		InitialStatus: session.AgentStatusStarting,
		IdleWindow:    cfg.Adapter.IdleWindow,
	})

	sup := &supervisor{
		cfg:            cfg,
		ctx:            ctx,
		decision:       decision,
		signals:        make(chan Signal, 64),
		paneSink:       make(chan observer.Snapshot, 4),
		statusSnapshot: session.AgentStatusStarting,
	}

	// Observer registration — optional but cheap. Only worth registering if
	// the adapter has at least one phrase to match against the pane.
	if cfg.Observer != nil && cfg.PaneTarget != "" && len(cfg.Adapter.AwaitingInputPhrases) > 0 {
		sup.paneCancel = cfg.Observer.Register(observer.Pane{
			SessionID: cfg.SessionID,
			Target:    cfg.PaneTarget,
			Sink:      sup.paneSink,
		})
	}

	sup.wg.Add(1)
	go sup.decisionLoop()
	sup.wg.Add(1)
	go sup.transcriptLoop()
	sup.wg.Add(1)
	go sup.livenessLoop()
	sup.wg.Add(1)
	go sup.paneLoop()

	return func() {
		// Unregister the pane observer FIRST so it stops publishing into
		// paneSink before the decision loop drains; only then cancel the
		// context so producer goroutines exit cleanly.
		if sup.paneCancel != nil {
			sup.paneCancel()
		}
		cancelCtx()
		sup.wg.Wait()
	}, nil
}

type missingFieldError string

func (e missingFieldError) Error() string { return "agent: supervisor config missing " + string(e) }

func errMissing(field string) error { return missingFieldError(field) }

// supervisor holds the runtime state for one session.
type supervisor struct {
	cfg      SupervisorConfig
	ctx      context.Context
	decision *Decision

	signals    chan Signal
	paneSink   chan observer.Snapshot
	paneCancel func()

	wg sync.WaitGroup

	statusMu       sync.Mutex // guards statusSnapshot
	statusSnapshot session.AgentStatus
}

func (s *supervisor) decisionLoop() {
	defer s.wg.Done()
	publish := func(trans Transition) {
		s.statusMu.Lock()
		s.statusSnapshot = trans.Status
		s.statusMu.Unlock()
		s.cfg.Publisher(trans)
	}

	for {
		select {
		case <-s.ctx.Done():
			return
		case sig := <-s.signals:
			status, changed := s.decision.Apply(sig)
			if !changed {
				continue
			}
			publish(Transition{
				SessionID: s.cfg.SessionID,
				TicketID:  s.cfg.TicketID,
				Status:    status,
				Tool:      s.decision.Tool(),
				Work:      s.decision.Work(),
				At:        sig.At,
			})
		}
	}
}

func (s *supervisor) transcriptLoop() {
	defer s.wg.Done()
	if s.cfg.Adapter.ResolveTranscript == nil || s.cfg.Adapter.ParseTranscriptLine == nil {
		return
	}

	// Phase 1: discovery — poll ResolveTranscript until it returns a path or
	// DiscoveryTimeout elapses. Liveness disappearance exits early.
	deadline := time.Time{}
	if s.cfg.Adapter.DiscoveryTimeout > 0 {
		deadline = s.cfg.now().Add(s.cfg.Adapter.DiscoveryTimeout)
	}
	var transcriptPath string
	for {
		if _, err := os.Stat(s.cfg.LivenessPath); os.IsNotExist(err) {
			return
		}
		if p := s.cfg.Adapter.ResolveTranscript(s.cfg.Runtime); p != "" {
			transcriptPath = p
			break
		}
		if !deadline.IsZero() && s.cfg.now().After(deadline) {
			s.cfg.Logger.Warn("agent supervisor: discovery_timeout",
				"session_id", s.cfg.SessionID,
				"ticket_id", s.cfg.TicketID,
				"agent", s.cfg.Adapter.Name,
			)
			s.send(Signal{Source: SourceTranscript, IsError: true, At: s.cfg.now()})
			return
		}
		if !sleep(s.ctx, s.cfg.DiscoveryInterval) {
			return
		}
	}

	f, err := os.Open(transcriptPath)
	if err != nil {
		s.cfg.Logger.Warn("agent supervisor: transcript open failed",
			"error", err, "path", transcriptPath, "session_id", s.cfg.SessionID)
		return
	}
	defer func() { _ = f.Close() }()
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		s.cfg.Logger.Warn("agent supervisor: transcript seek failed",
			"error", err, "path", transcriptPath, "session_id", s.cfg.SessionID)
		return
	}

	reader := bufio.NewScanner(f)
	reader.Buffer(make([]byte, 64*1024), transcriptLineBufferMaxLen)

	for {
		for reader.Scan() {
			line := reader.Bytes()
			if len(line) == 0 {
				continue
			}
			ev := s.cfg.Adapter.ParseTranscriptLine(line)
			sig := Signal{Source: SourceTranscript, At: s.cfg.now()}
			sig.Activity = ev.Activity
			sig.Status = ev.Status
			sig.IsError = ev.IsError
			if ev.Tool != "" {
				t := ev.Tool
				sig.Tool = &t
			}
			if ev.Work != "" {
				w := ev.Work
				sig.Work = &w
			}
			// Drop the signal if nothing useful is in it — keeps the decision
			// loop from doing busywork on blank lines.
			if !sig.Activity && sig.Status == "" && !sig.IsError && sig.Tool == nil && sig.Work == nil {
				continue
			}
			s.send(sig)
		}
		if err := reader.Err(); err != nil {
			// ErrTooLong: a single transcript line exceeded the 1 MiB cap.
			// Skip past it and reopen the scanner at the current offset so we
			// don't lose the rest of the transcript.
			s.cfg.Logger.Warn("agent supervisor: transcript scanner error",
				"error", err, "session_id", s.cfg.SessionID)
			reader = bufio.NewScanner(f)
			reader.Buffer(make([]byte, 64*1024), transcriptLineBufferMaxLen)
			continue
		}
		if _, err := os.Stat(s.cfg.LivenessPath); os.IsNotExist(err) {
			return
		}
		if !sleep(s.ctx, s.cfg.FollowInterval) {
			return
		}
		reader = bufio.NewScanner(f)
		reader.Buffer(make([]byte, 64*1024), transcriptLineBufferMaxLen)
	}
}

func (s *supervisor) livenessLoop() {
	defer s.wg.Done()
	for {
		if !sleep(s.ctx, s.cfg.LivenessInterval) {
			return
		}
		if _, err := os.Stat(s.cfg.LivenessPath); os.IsNotExist(err) {
			s.send(Signal{Source: SourceLiveness, At: s.cfg.now()})
			return
		}
	}
}

// paneLoop translates observer snapshots into decision signals. Each tick
// produces one Signal; substring match against AwaitingInputPhrases is the
// only pane-derived disambiguation the decision machine gets. The observer's
// per-tick heartbeat doubles as the idle-timer drive — no separate timer
// goroutine required.
func (s *supervisor) paneLoop() {
	defer s.wg.Done()
	for {
		select {
		case <-s.ctx.Done():
			return
		case snap, ok := <-s.paneSink:
			if !ok {
				return
			}
			sig := Signal{
				Source:  SourcePane,
				At:      snap.At,
				Changed: snap.Changed,
			}
			if phrase := s.cfg.Adapter.MatchAwaitingInput(snap.Content); phrase != "" {
				sig.HasAwaitingPhrase = true
				RecordPhraseHit(s.cfg.Adapter.Name, phrase)
			}
			s.send(sig)
		}
	}
}

func (s *supervisor) send(sig Signal) {
	// Non-blocking send: if the decision loop has already returned and the
	// channel is full, drop the signal rather than leak the producer
	// goroutine. Under normal operation signals is buffered (64) and the
	// decision loop drains fast enough that the default branch is rare.
	select {
	case <-s.ctx.Done():
	case s.signals <- sig:
	default:
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
// /agent/status endpoint. daemonURL and architectPath are captured by
// closure so callers don't need to thread them per-call. Failures are
// logged but not retried — the next transition carries the full state, so
// one dropped call causes at most one frame of stale UI.
func HTTPPublisher(daemonURL, architectPath string) Publisher {
	return HTTPPublisherWithLogger(daemonURL, architectPath, nil)
}

// HTTPPublisherWithLogger is HTTPPublisher with an injectable logger. The
// supervisor uses this internally so transition drops surface under the
// session's log attributes.
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
