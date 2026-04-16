package spawn

import "encoding/json"

// parseOpenCodeLine decodes one JSONL line written by the opencode status
// plugin. The plugin emits `{"status":"...","tool":"..."}` on every event;
// the tailer forwards it verbatim to /agent/status.
func parseOpenCodeLine(line []byte) StatusUpdate {
	var u StatusUpdate
	if err := json.Unmarshal(line, &u); err != nil {
		return StatusUpdate{}
	}
	return u
}

// StartOpenCodeTailer starts a background goroutine that tails the per-spawn
// status jsonl file written by the opencode status plugin. livenessPath is
// the MCP config temp file — the launcher EXIT trap removes it when opencode
// exits, which signals the tailer to stop (same pattern as codex/claude).
//
// If ticketID, statusFilePath, or livenessPath is empty the call is a no-op.
func StartOpenCodeTailer(statusFilePath, livenessPath, ticketID, architectPath, daemonURL string) {
	StartStatusTailer(TailerConfig{
		ResolveTranscript: ResolveFixedPath(statusFilePath),
		LivenessPath:      livenessPath,
		TicketID:          ticketID,
		ArchitectPath:     architectPath,
		DaemonURL:         daemonURL,
		Parser:            parseOpenCodeLine,
	})
}
