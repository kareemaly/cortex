package spawn

import (
	"encoding/json"
	"path/filepath"
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

// parseRolloutLine parses a single jsonl line and returns the agent status
// that should be POSTed to /agent/status, or an empty StatusUpdate if the
// line should be ignored.
func parseRolloutLine(line []byte) StatusUpdate {
	var rl rolloutLine
	if err := json.Unmarshal(line, &rl); err != nil {
		return StatusUpdate{}
	}
	switch rl.Type {
	case "session_meta":
		// First line written by codex — process is up and accepting input.
		return StatusUpdate{Status: "idle"}
	case "event_msg":
		var p rolloutPayload
		if err := json.Unmarshal(rl.Payload, &p); err != nil {
			return StatusUpdate{}
		}
		switch p.Type {
		case "task_started":
			return StatusUpdate{Status: "in_progress"}
		case "task_complete":
			return StatusUpdate{Status: "idle"}
		}
	}
	return StatusUpdate{}
}

// StartCodexTailer starts a background goroutine that tails the codex rollout
// jsonl file under codexHome and posts agent status updates to the cortexd
// daemon. codexHome doubles as the liveness marker — the launcher EXIT trap
// runs `rm -rf $CODEX_HOME` when codex exits, which signals the tailer to
// stop.
//
// If ticketID or codexHome is empty the call is a no-op (collab sessions
// don't carry a ticket_id; status wiring for collab is a future ticket).
func StartCodexTailer(codexHome, ticketID, architectPath, daemonURL string) {
	if codexHome == "" {
		return
	}
	pattern := filepath.Join(codexHome, "sessions", "*", "*", "*", "rollout-*.jsonl")
	StartStatusTailer(TailerConfig{
		ResolveTranscript: func() string {
			matches, err := filepath.Glob(pattern)
			if err != nil || len(matches) == 0 {
				return ""
			}
			return matches[0]
		},
		LivenessPath:  codexHome,
		TicketID:      ticketID,
		ArchitectPath: architectPath,
		DaemonURL:     daemonURL,
		Parser:        parseRolloutLine,
	})
}
