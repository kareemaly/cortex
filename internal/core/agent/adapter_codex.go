package agent

import (
	"encoding/json"
	"errors"
	"io/fs"
	"path/filepath"
	"regexp"
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

// parseCodexLine interprets codex rollout events: session_meta is a
// live-but-quiet signal (seed idle); event_msg payloads drive
// working/idle transitions.
func parseCodexLine(line []byte) StatusUpdate {
	var rl rolloutLine
	if err := json.Unmarshal(line, &rl); err != nil {
		return StatusUpdate{}
	}
	switch rl.Type {
	case "session_meta":
		return StatusUpdate{Status: session.AgentStatusIdle}
	case "event_msg":
		var p rolloutPayload
		if err := json.Unmarshal(rl.Payload, &p); err != nil {
			return StatusUpdate{}
		}
		switch p.Type {
		case "task_started":
			return StatusUpdate{Status: session.AgentStatusWorking}
		case "task_complete":
			return StatusUpdate{Status: session.AgentStatusIdle}
		}
	}
	return StatusUpdate{}
}

// findCodexRollout walks codexHome (bounded) looking for the rollout
// jsonl codex writes per session. Codex has reorganized this directory
// layout before, so a capped recursive walk is more durable than a
// fixed-depth glob.
//
// Returns "" when no rollout is found; the supervisor re-polls.
func findCodexRollout(codexHome string) string {
	if codexHome == "" {
		return ""
	}
	sessions := filepath.Join(codexHome, "sessions")
	var found string
	// walkDir bails on the first rollout-*.jsonl hit. Guard the descent
	// depth so a misconfigured codex home doesn't turn discovery into a
	// full-disk walk.
	_ = filepath.WalkDir(sessions, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return fs.SkipDir
			}
			return nil
		}
		// Depth guard: sessions/<year>/<month>/<day>/rollout-*.jsonl is
		// the current layout; anything 6+ deep is almost certainly a
		// misconfigured home and not worth descending.
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

// codexApprovalBox matches codex's approval-prompt shape. The structural
// marker is a `^> ` prompt preceded by `│`-indented body lines; the
// anchor matches the y/n / approval language. Border-AND-anchor keeps
// the regex from firing on stray `> ` lines in tool output.
var codexApprovalBox = &BoxPattern{
	Name:    "codex_approval",
	Border:  regexp.MustCompile(`(?m)^│`),
	Anchor:  regexp.MustCompile(`(?mi)(\(y\/n\)|requires approval|allow this command)`),
	Implies: session.AgentStatusAwaitingInput,
}

var codexAdapter = &Adapter{
	Name:             "codex",
	IdleThreshold:    0, // codex emits task_complete explicitly
	DiscoveryTimeout: 30 * time.Second,

	ResolveTranscript: func(rt RuntimeCtx) string {
		// TranscriptHint, when set by Prepare, is the per-session codex
		// home directory. When unset (legacy), use WorkingDir — which the
		// spawner should populate with CODEX_HOME.
		home := rt.TranscriptHint
		if home == "" {
			home = rt.Env["CODEX_HOME"]
		}
		return findCodexRollout(home)
	},
	ParseLine: parseCodexLine,

	PanePatterns: PanePatterns{
		SearchTailLines: 14,
		Boxes:           []*BoxPattern{codexApprovalBox},
	},
}

func init() { Register(codexAdapter) }
