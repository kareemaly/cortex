package spawn

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"
)

// Default timing used by all agent tailers. Picked to match the original
// codex tailer so behavior is unchanged for existing agents.
const (
	tailerDiscoveryInterval = 250 * time.Millisecond
	tailerFollowInterval    = 100 * time.Millisecond
	tailerDiscoveryTimeout  = 30 * time.Second
	tailerLineBufferMax     = 1 << 20 // 1 MB — matches codex session_meta worst case.
)

// StatusUpdate is what a per-agent parser returns for one transcript line.
// Status "" means "ignore this line". Tool and Work are forwarded to the
// daemon verbatim; leave them empty when not applicable.
type StatusUpdate struct {
	Status string `json:"status"`
	Tool   string `json:"tool,omitempty"`
	Work   string `json:"work,omitempty"`
}

// StatusParser turns a single transcript line into a StatusUpdate.
type StatusParser func(line []byte) StatusUpdate

// TailerConfig configures a single transcript tailer goroutine.
//
// The tailer runs three phases:
//
//  1. Discovery — calls ResolveTranscript repeatedly (every DiscoveryInterval)
//     until it returns a non-empty path or DiscoveryTimeout elapses. Polling
//     stops early if LivenessPath disappears (agent exited before writing).
//
//  2. Follow — opens the transcript, seeks to EOF (skipping historical
//     content on resume), reads appended lines with a bufio.Scanner, feeds
//     each line to Parser and POSTs the resulting status transitions to
//     DaemonURL + "/agent/status". At every EOF it re-checks LivenessPath
//     and exits cleanly when the marker disappears.
//
//  3. Idle (optional) — when IdleThreshold > 0, the follow loop flips the
//     current status to "idle" once no new line has arrived for that long.
//     Used by the claude tailer, which has no explicit "turn complete"
//     signal in its transcript.
//
// InitialStatus is POSTed once after the transcript file opens, before any
// lines are read. Leave empty to skip (e.g. codex's first transcript line is
// already session_meta → idle, so no seed is needed).
type TailerConfig struct {
	ResolveTranscript func() string
	LivenessPath      string
	TicketID          string
	ArchitectPath     string
	DaemonURL         string
	Parser            StatusParser
	InitialStatus     string
	IdleThreshold     time.Duration
	DiscoveryTimeout  time.Duration
	DiscoveryInterval time.Duration
	FollowInterval    time.Duration
}

// ResolveFixedPath is the common ResolveTranscript helper for tailers whose
// transcript path is fully known at spawn time (claude, opencode). Returns
// the path once os.Stat succeeds, "" otherwise.
func ResolveFixedPath(path string) func() string {
	return func() string {
		if path == "" {
			return ""
		}
		if _, err := os.Stat(path); err == nil {
			return path
		}
		return ""
	}
}

// StartStatusTailer launches the shared tailer goroutine. No-op if any
// required input is missing — callers don't need to gate their own calls.
func StartStatusTailer(cfg TailerConfig) {
	if cfg.TicketID == "" || cfg.LivenessPath == "" || cfg.ResolveTranscript == nil || cfg.Parser == nil {
		return
	}
	if cfg.DaemonURL == "" {
		return
	}
	if cfg.DiscoveryInterval == 0 {
		cfg.DiscoveryInterval = tailerDiscoveryInterval
	}
	if cfg.FollowInterval == 0 {
		cfg.FollowInterval = tailerFollowInterval
	}
	if cfg.DiscoveryTimeout == 0 {
		cfg.DiscoveryTimeout = tailerDiscoveryTimeout
	}
	go runStatusTailer(cfg)
}

func runStatusTailer(cfg TailerConfig) {
	client := &http.Client{Timeout: 5 * time.Second}

	postStatus := func(u StatusUpdate) {
		if u.Status == "" {
			return
		}
		payload := map[string]any{
			"ticket_id": cfg.TicketID,
			"status":    u.Status,
		}
		if u.Tool != "" {
			payload["tool"] = u.Tool
		}
		if u.Work != "" {
			payload["work"] = u.Work
		}
		body, _ := json.Marshal(payload)
		req, err := http.NewRequest(http.MethodPost, cfg.DaemonURL+"/agent/status", bytes.NewReader(body))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Cortex-Architect", cfg.ArchitectPath)
		resp, err := client.Do(req)
		if err != nil {
			return
		}
		_ = resp.Body.Close()
	}

	// Phase 1: discovery.
	var transcriptPath string
	deadline := time.Now().Add(cfg.DiscoveryTimeout)
	for {
		if _, err := os.Stat(cfg.LivenessPath); os.IsNotExist(err) {
			return
		}
		if p := cfg.ResolveTranscript(); p != "" {
			transcriptPath = p
			break
		}
		if time.Now().After(deadline) {
			return
		}
		time.Sleep(cfg.DiscoveryInterval)
	}

	f, err := os.Open(transcriptPath)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()

	// Skip any historical content — we only care about lines written by the
	// current session. For fresh spawns the file is empty so this is a no-op.
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return
	}

	newScanner := func() *bufio.Scanner {
		s := bufio.NewScanner(f)
		s.Buffer(make([]byte, 64*1024), tailerLineBufferMax)
		return s
	}

	currentStatus := ""
	if cfg.InitialStatus != "" {
		postStatus(StatusUpdate{Status: cfg.InitialStatus})
		currentStatus = cfg.InitialStatus
	}

	lastLineAt := time.Now()

	scanner := newScanner()
	for {
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}
			lastLineAt = time.Now()
			update := cfg.Parser(line)
			if update.Status == "" {
				continue
			}
			if update.Status != currentStatus || update.Tool != "" || update.Work != "" {
				postStatus(update)
				currentStatus = update.Status
			}
		}

		if err := scanner.Err(); err != nil && err != io.EOF {
			return
		}

		if _, err := os.Stat(cfg.LivenessPath); os.IsNotExist(err) {
			return
		}

		if cfg.IdleThreshold > 0 && currentStatus != "idle" && currentStatus != "" && time.Since(lastLineAt) >= cfg.IdleThreshold {
			postStatus(StatusUpdate{Status: "idle"})
			currentStatus = "idle"
		}

		time.Sleep(cfg.FollowInterval)
		scanner = newScanner()
	}
}
