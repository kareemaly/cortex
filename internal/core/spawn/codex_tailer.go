package spawn

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// rolloutLine is the top-level shape of each jsonl line in the codex rollout file.
type rolloutLine struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// rolloutPayload extracts the nested type discriminator from event_msg payloads.
type rolloutPayload struct {
	Type string `json:"type"`
}

// parseRolloutLine parses a single jsonl line and returns the agent status string
// that should be POSTed to /agent/status, or "" if the line should be ignored.
func parseRolloutLine(line []byte) string {
	var rl rolloutLine
	if err := json.Unmarshal(line, &rl); err != nil {
		return ""
	}
	switch rl.Type {
	case "session_meta":
		// First line written by codex — process is up and accepting input.
		return "idle"
	case "event_msg":
		var p rolloutPayload
		if err := json.Unmarshal(rl.Payload, &p); err != nil {
			return ""
		}
		switch p.Type {
		case "task_started":
			return "in_progress"
		case "task_complete":
			return "idle"
		}
	}
	return ""
}

// StartCodexTailer starts a background goroutine that tails the codex rollout
// jsonl file in codexHome and posts agent status updates to the cortexd daemon.
//
// The goroutine self-terminates when codexHome is removed — the launcher EXIT
// trap runs `rm -rf $CODEX_HOME` when the codex process exits.
//
// If ticketID or codexHome is empty the call is a no-op (collab sessions do not
// have a ticket_id; status wiring for collab is deferred to a future ticket).
func StartCodexTailer(codexHome, ticketID, architectPath, daemonURL string) {
	if ticketID == "" || codexHome == "" {
		return
	}
	go runCodexTailer(codexHome, ticketID, architectPath, daemonURL)
}

func runCodexTailer(codexHome, ticketID, architectPath, daemonURL string) {
	client := &http.Client{Timeout: 5 * time.Second}

	postStatus := func(status string) {
		body, _ := json.Marshal(map[string]string{
			"ticket_id": ticketID,
			"status":    status,
		})
		req, err := http.NewRequest(http.MethodPost, daemonURL+"/agent/status", bytes.NewReader(body))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Cortex-Architect", architectPath)
		resp, err := client.Do(req)
		if err != nil {
			return
		}
		_ = resp.Body.Close()
	}

	// Phase 1: wait for the rollout file to appear under codexHome/sessions/*/*/*/rollout-*.jsonl
	pattern := filepath.Join(codexHome, "sessions", "*", "*", "*", "rollout-*.jsonl")
	var rolloutPath string
	for {
		if _, err := os.Stat(codexHome); os.IsNotExist(err) {
			return // codexHome cleaned up before file appeared
		}
		matches, err := filepath.Glob(pattern)
		if err == nil && len(matches) > 0 {
			rolloutPath = matches[0]
			break
		}
		time.Sleep(250 * time.Millisecond)
	}

	// Phase 2: open file and read in follow-mode (tail -f style).
	f, err := os.Open(rolloutPath)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()

	newScanner := func() *bufio.Scanner {
		s := bufio.NewScanner(f)
		s.Buffer(make([]byte, 64*1024), 1<<20) // 1 MB max — session_meta embeds full system prompt
		return s
	}

	scanner := newScanner()
	for {
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}
			if status := parseRolloutLine(line); status != "" {
				postStatus(status)
			}
		}

		if err := scanner.Err(); err != nil && err != io.EOF {
			return // unexpected scanner error
		}

		// At EOF — check whether the process has exited (codexHome removed by EXIT trap).
		if _, err := os.Stat(codexHome); os.IsNotExist(err) {
			return
		}

		// Process still alive; wait briefly for new lines then continue reading.
		// Recreate the scanner on the same file handle — it retains its position,
		// so the new scanner picks up only newly appended bytes.
		time.Sleep(100 * time.Millisecond)
		scanner = newScanner()
	}
}
