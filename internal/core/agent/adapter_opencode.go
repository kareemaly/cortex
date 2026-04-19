package agent

import (
	"encoding/json"
	"os"
	"time"

	"github.com/kareemaly/cortex/internal/session"
)

// opencodePluginLine mirrors the shape the opencode status plugin emits on
// every event: {"status": "...", "tool": "...", "work": "..."}.
type opencodePluginLine struct {
	Status string `json:"status"`
	Tool   string `json:"tool,omitempty"`
	Work   string `json:"work,omitempty"`
}

// parseOpenCodeLine forwards the plugin payload verbatim. The plugin emits
// canonical status names (working/idle/awaiting_input/error) so no
// translation is needed — the decision machine treats plugin status as
// authoritative.
func parseOpenCodeLine(line []byte) TranscriptEvent {
	var p opencodePluginLine
	if err := json.Unmarshal(line, &p); err != nil {
		return TranscriptEvent{}
	}
	return TranscriptEvent{
		Status: session.AgentStatus(p.Status),
		Tool:   p.Tool,
		Work:   p.Work,
	}
}

var opencodeAdapter = &Adapter{
	Name: "opencode",
	// Plugin pushes status authoritatively. IdleWindow is only a fallback if
	// the plugin wedges (shouldn't happen, but it's cheap insurance).
	IdleWindow:       5 * time.Second,
	DiscoveryTimeout: 30 * time.Second,

	// Backup only — the plugin is primary. If the plugin hiccups we can still
	// surface the waiting-for-permission state from the pane itself.
	AwaitingInputPhrases: []string{
		"Permission required to run this command",
	},

	ResolveTranscript: func(rt RuntimeCtx) string {
		if rt.TranscriptHint == "" {
			return ""
		}
		if _, err := os.Stat(rt.TranscriptHint); err == nil {
			return rt.TranscriptHint
		}
		return ""
	},
	ParseTranscriptLine: parseOpenCodeLine,
}

func init() { Register(opencodeAdapter) }
