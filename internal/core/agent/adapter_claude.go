package agent

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/kareemaly/cortex/internal/session"
)

// claudeIdleThreshold matches the existing claude tailer. Chosen short
// enough that the dashboard feels live and long enough to avoid flapping
// during mid-turn pauses (waiting on a tool call).
const claudeIdleThreshold = 5 * time.Second

// ClaudeTranscriptPath returns the Claude Code transcript jsonl path for
// a given working directory and Claude session UUID. Claude writes
// transcripts to $HOME/.claude/projects/<slug>/<session-id>.jsonl, where
// <slug> is the absolute cwd with "/" replaced by "-" and a leading "-".
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

// claudeProjectSlug encodes the working directory the way claude-code
// expects on disk: absolute path, trailing slash stripped, "/" replaced
// with "-". The resulting slug is the projects/<slug>/ directory name.
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

// parseClaudeLine treats every non-empty transcript line as evidence of
// activity → working. The bug fix for working-while-permission lives in
// the decision machine (pane signal governs awaiting_input) plus the
// pane patterns below — not in ParseLine, which has no visibility into
// the dialog.
func parseClaudeLine(line []byte) StatusUpdate {
	if len(line) == 0 {
		return StatusUpdate{}
	}
	return StatusUpdate{Status: session.AgentStatusWorking}
}

// claudePermissionBox captures Claude Code's permission dialog. The
// border regex matches the ╭ top-edge or the ╰ bottom-edge of the box;
// the anchor combines the ❯ focus caret with one of the known options
// ("1. Yes" / "2. No" / "tell Claude what to do differently"). Requiring
// border AND anchor within the same tail window rules out chatter like
// "❯ Yes" in regular output.
var claudePermissionBox = &BoxPattern{
	Name:    "claude_permission",
	Border:  regexp.MustCompile(`(?m)^[╭╰]─`),
	Anchor:  regexp.MustCompile(`(?m)❯\s+\d+\.\s+(Yes|No)|tell Claude what to do differently`),
	Implies: session.AgentStatusAwaitingInput,
}

// claudeSelectBox is the /slash-command selector and model picker shape —
// same border style, caret anchor without the Yes/No answer text. We
// treat it as awaiting_input too since Claude is paused on the user.
var claudeSelectBox = &BoxPattern{
	Name:    "claude_select",
	Border:  regexp.MustCompile(`(?m)^[╭╰]─`),
	Anchor:  regexp.MustCompile(`(?m)^\s*│\s*❯\s+\S`),
	Implies: session.AgentStatusAwaitingInput,
}

var claudeAdapter = &Adapter{
	Name:             "claude",
	IdleThreshold:    claudeIdleThreshold,
	DiscoveryTimeout: 30 * time.Second,

	ResolveTranscript: func(rt RuntimeCtx) string {
		if rt.TranscriptHint != "" {
			if _, err := os.Stat(rt.TranscriptHint); err == nil {
				return rt.TranscriptHint
			}
		}
		return ""
	},
	ParseLine: parseClaudeLine,

	PanePatterns: PanePatterns{
		SearchTailLines: 14,
		Boxes:           []*BoxPattern{claudePermissionBox, claudeSelectBox},
	},
}

func init() { Register(claudeAdapter) }
