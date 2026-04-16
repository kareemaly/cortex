package spawn

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// claudeIdleThreshold is how long the tailer waits for silence on the
// transcript before flipping status back to idle. Claude Code emits no
// explicit "turn complete" event we can parse, so we derive idle from
// recent write activity. 5s is short enough to feel live on the dashboard
// and long enough to avoid flapping during mid-turn pauses (e.g. waiting
// on a tool call).
const claudeIdleThreshold = 5 * time.Second

// ClaudeTranscriptPath returns the Claude Code transcript jsonl path for a
// given working directory and session UUID. Claude writes transcripts to
// $HOME/.claude/projects/<slug>/<session-id>.jsonl, where <slug> is the
// absolute cwd with "/" replaced by "-" and a leading "-".
//
// Returns "" if cwd or sessionID is empty, or if the home directory cannot
// be resolved.
func ClaudeTranscriptPath(cwd, sessionID string) string {
	if cwd == "" || sessionID == "" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "projects", claudeProjectSlug(cwd), sessionID+".jsonl")
}

// claudeProjectSlug returns the Claude Code projects-directory slug for cwd.
// "/Users/foo/bar" -> "-Users-foo-bar". Trailing slashes are trimmed.
func claudeProjectSlug(cwd string) string {
	if cwd == "" {
		return ""
	}
	abs, err := filepath.Abs(cwd)
	if err != nil {
		abs = cwd
	}
	abs = strings.TrimRight(abs, "/")
	return strings.ReplaceAll(abs, "/", "-")
}

// parseClaudeLine treats every non-empty transcript line as recent activity
// and flips the session to in_progress. Idle transitions are handled by the
// shared tailer's IdleThreshold, not by parsing specific claude event types.
func parseClaudeLine(line []byte) StatusUpdate {
	if len(line) == 0 {
		return StatusUpdate{}
	}
	return StatusUpdate{Status: "in_progress"}
}

// StartClaudeTailer starts a background goroutine that tails the Claude Code
// transcript at transcriptPath. livenessPath is the MCP config temp file
// written by the launcher — the EXIT trap removes it when claude exits, which
// signals the tailer to stop (same pattern as codex's CODEX_HOME liveness).
//
// If ticketID, transcriptPath, or livenessPath is empty the call is a no-op.
func StartClaudeTailer(transcriptPath, livenessPath, ticketID, architectPath, daemonURL string) {
	StartStatusTailer(TailerConfig{
		ResolveTranscript: ResolveFixedPath(transcriptPath),
		LivenessPath:      livenessPath,
		TicketID:          ticketID,
		ArchitectPath:     architectPath,
		DaemonURL:         daemonURL,
		Parser:            parseClaudeLine,
		InitialStatus:     "idle",
		IdleThreshold:     claudeIdleThreshold,
	})
}
