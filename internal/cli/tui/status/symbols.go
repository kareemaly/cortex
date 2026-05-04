// Package status centralizes agent-status symbol and styling rendering
// so kanban and dashboard views stay in lockstep. All callers pass the
// canonical status string (see session.AgentStatus).
package status

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/kareemaly/cortex/internal/types"
)

// OrphanedIcon is rendered when a session tracking marker exists but the
// process has gone away (see ticket orphan detection).
const OrphanedIcon = "◌"

// defaultIcon is returned when status is missing or unrecognized.
const defaultIcon = "●"

// symbols maps canonical agent-status names to display icons.
var symbols = map[string]string{
	"starting":       "▶",
	"working":        "●",
	"idle":           "○",
	"awaiting_input": "⏸",
	"error":          "✗",
	"ended":          "○",
}

// endedStyle is applied on top of the caller's base style when rendering
// the "ended" state so the row stays visible but reads as informational.
var endedStyle = lipgloss.NewStyle().Faint(true)

// Icon returns the glyph for the given status. Empty or unknown status
// falls back to the default icon.
func Icon(s string) string {
	if sym, ok := symbols[s]; ok {
		return sym
	}
	return defaultIcon
}

// IsEnded reports whether the status represents a completed/gone session
// that should render dimmed rather than active.
func IsEnded(s string) bool {
	return s == "ended"
}

// ApplyEnded wraps a rendered cell in the ended style when appropriate,
// leaving it untouched otherwise.
func ApplyEnded(status, rendered string) string {
	if IsEnded(status) {
		return endedStyle.Render(rendered)
	}
	return rendered
}

// TicketIcon returns the status icon for a ticket summary, falling back to
// the orphaned icon when the session has been orphaned.
func TicketIcon(t types.TicketSummary) string {
	if t.IsOrphaned {
		return OrphanedIcon
	}
	s := ""
	if t.AgentStatus != nil {
		s = *t.AgentStatus
	}
	return Icon(s)
}
