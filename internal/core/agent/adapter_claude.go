package agent

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ClaudeTranscriptPath returns the Claude Code transcript jsonl path for a
// given working directory and Claude session UUID. Claude writes transcripts
// to $HOME/.claude/projects/<slug>/<session-id>.jsonl, where <slug> is the
// absolute cwd with "/" replaced by "-".
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

// claudeProjectSlug encodes the working directory the way claude-code expects
// on disk: absolute path, trailing slash stripped, "/" replaced with "-".
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

// parseClaudeLine treats every non-empty transcript line as activity. Claude
// has no explicit "task complete" marker in its JSONL — idle detection is
// elapsed-time-driven in the decision machine.
func parseClaudeLine(line []byte) TranscriptEvent {
	if len(line) == 0 {
		return TranscriptEvent{}
	}
	return TranscriptEvent{Activity: true}
}

var claudeAdapter = &Adapter{
	Name:             "claude",
	DiscoveryTimeout: 30 * time.Second,

	ResolveTranscript: func(rt RuntimeCtx) string {
		if rt.TranscriptHint != "" {
			if _, err := os.Stat(rt.TranscriptHint); err == nil {
				return rt.TranscriptHint
			}
		}
		return ""
	},
	ParseTranscriptLine: parseClaudeLine,
}

func init() { Register(claudeAdapter) }
