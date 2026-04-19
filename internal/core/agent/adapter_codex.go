package agent

import (
	"encoding/json"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/kareemaly/cortex/internal/session"
)

// rolloutLine is the top-level shape of each jsonl line codex emits.
type rolloutLine struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type rolloutPayload struct {
	Type string `json:"type"`
}

// parseCodexLine interprets codex rollout events. Codex's JSONL is
// authoritative: task_started/task_complete say exactly what the agent is
// doing; session_meta is a "connected but idle" seed. Pane phrase matching
// handles approval dialogs separately (codex prompts on-screen only).
func parseCodexLine(line []byte) TranscriptEvent {
	var rl rolloutLine
	if err := json.Unmarshal(line, &rl); err != nil {
		return TranscriptEvent{}
	}
	switch rl.Type {
	case "session_meta":
		return TranscriptEvent{Status: session.AgentStatusIdle}
	case "event_msg":
		var p rolloutPayload
		if err := json.Unmarshal(rl.Payload, &p); err != nil {
			return TranscriptEvent{}
		}
		switch p.Type {
		case "task_started":
			return TranscriptEvent{Status: session.AgentStatusWorking}
		case "task_complete":
			return TranscriptEvent{Status: session.AgentStatusIdle}
		}
	}
	return TranscriptEvent{}
}

// findCodexRollout walks codexHome (bounded) looking for the rollout jsonl
// codex writes per session. Codex has reorganized this directory layout
// before, so a capped recursive walk is more durable than a fixed-depth
// glob. Returns "" when no rollout is found; the supervisor re-polls.
func findCodexRollout(codexHome string) string {
	if codexHome == "" {
		return ""
	}
	sessions := filepath.Join(codexHome, "sessions")
	var found string
	// walkDir bails on the first rollout-*.jsonl hit. Guard the descent depth
	// so a misconfigured codex home doesn't turn discovery into a full-disk
	// walk.
	_ = filepath.WalkDir(sessions, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return fs.SkipDir
			}
			return nil
		}
		// Depth guard: sessions/<year>/<month>/<day>/rollout-*.jsonl is the
		// current layout; anything 6+ deep is almost certainly a misconfigured
		// home and not worth descending.
		rel, _ := filepath.Rel(sessions, path)
		if rel != "." && strings.Count(rel, string(filepath.Separator)) > 6 {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !d.IsDir() && strings.HasPrefix(d.Name(), "rollout-") && strings.HasSuffix(d.Name(), ".jsonl") {
			found = path
			return fs.SkipAll
		}
		return nil
	})
	return found
}

var codexAdapter = &Adapter{
	Name: "codex",
	// Codex emits task_started/task_complete explicitly, so transcript
	// status is authoritative and IdleWindow is only a fallback for long
	// gaps between transcript events with no pane change.
	IdleWindow:       3 * time.Second,
	DiscoveryTimeout: 30 * time.Second,

	// Codex approval prompts render "Allow this command?" verbatim. Narrower
	// than matching on "(y/n)", which could collide with content inside
	// tool stdout (e.g. an installer's own confirmation prompt echoed back).
	AwaitingInputPhrases: []string{
		"Allow this command?",
	},

	ResolveTranscript: func(rt RuntimeCtx) string {
		// TranscriptHint, when set by Prepare, is the per-session codex home
		// directory. When unset (legacy), use the CODEX_HOME env.
		home := rt.TranscriptHint
		if home == "" {
			home = rt.Env["CODEX_HOME"]
		}
		return findCodexRollout(home)
	},
	ParseTranscriptLine: parseCodexLine,
}

func init() { Register(codexAdapter) }
