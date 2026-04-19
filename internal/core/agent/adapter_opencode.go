package agent

import (
	"encoding/json"
	"os"
	"regexp"
	"time"

	"github.com/kareemaly/cortex/internal/session"
)

// opencodePluginLine mirrors the shape the opencode status plugin emits
// on every event: {"status": "...", "tool": "...", "work": "..."}.
type opencodePluginLine struct {
	Status string `json:"status"`
	Tool   string `json:"tool,omitempty"`
	Work   string `json:"work,omitempty"`
}

// parseOpenCodeLine forwards the plugin payload verbatim. The plugin emits
// canonical status names (working/idle/awaiting_input/error) so no translation
// is needed — unknown values flow through and the decision machine ignores them.
func parseOpenCodeLine(line []byte) StatusUpdate {
	var p opencodePluginLine
	if err := json.Unmarshal(line, &p); err != nil {
		return StatusUpdate{}
	}
	return StatusUpdate{
		Status: session.AgentStatus(p.Status),
		Tool:   p.Tool,
		Work:   p.Work,
	}
}

// opencodePermissionBox is a defensive fallback — the plugin is the
// primary signal source, but if it hiccups we can still surface the
// waiting-for-permission state from the pane itself.
var opencodePermissionBox = &BoxPattern{
	Name:    "opencode_permission",
	Border:  regexp.MustCompile(`(?m)^─{3,}`),
	Anchor:  regexp.MustCompile(`(?i)(permission|allow|deny)`),
	Implies: session.AgentStatusAwaitingInput,
}

var opencodeAdapter = &Adapter{
	Name:             "opencode",
	IdleThreshold:    0, // plugin reports idle explicitly
	DiscoveryTimeout: 30 * time.Second,

	ResolveTranscript: func(rt RuntimeCtx) string {
		if rt.TranscriptHint == "" {
			return ""
		}
		if _, err := os.Stat(rt.TranscriptHint); err == nil {
			return rt.TranscriptHint
		}
		return ""
	},
	ParseLine: parseOpenCodeLine,

	PanePatterns: PanePatterns{
		SearchTailLines: 14,
		Boxes:           []*BoxPattern{opencodePermissionBox},
	},
}

func init() { Register(opencodeAdapter) }
