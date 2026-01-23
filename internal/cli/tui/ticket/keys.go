package ticket

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// Key represents a keyboard key.
type Key string

// Key constants for navigation and actions.
const (
	KeyQuit        Key = "q"
	KeyCtrlC       Key = "ctrl+c"
	KeyUp          Key = "up"
	KeyDown        Key = "down"
	KeyK           Key = "k"
	KeyJ           Key = "j"
	KeyRefresh     Key = "r"
	KeyPgUp        Key = "pgup"
	KeyPgDown      Key = "pgdown"
	KeyHome        Key = "home"
	KeyEnd         Key = "end"
	KeyKillSession Key = "x"
	KeyYes         Key = "y"
	KeyNo          Key = "n"
	KeyEscape      Key = "esc"
)

// isKey checks if a key message matches a key constant.
func isKey(msg tea.KeyMsg, keys ...Key) bool {
	for _, k := range keys {
		if msg.String() == string(k) {
			return true
		}
	}
	return false
}

// helpText returns the help bar text for the ticket detail view.
func helpText(scrollPercent int, hasActiveSession bool) string {
	base := "[j/k] scroll  [r]efresh  [q]uit"
	if hasActiveSession {
		base = "[j/k] scroll  [r]efresh  [x] kill session  [q]uit"
	}
	return base + "  " + percentStr(scrollPercent)
}

// percentStr formats a scroll percentage string.
func percentStr(percent int) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	return fmt.Sprintf("%3d%%", percent)
}
