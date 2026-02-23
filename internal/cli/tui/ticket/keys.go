package ticket

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// Key represents a keyboard key.
type Key string

// Key constants for navigation and actions.
const (
	KeyQuit         Key = "q"
	KeyCtrlC        Key = "ctrl+c"
	KeyUp           Key = "up"
	KeyDown         Key = "down"
	KeyK            Key = "k"
	KeyJ            Key = "j"
	KeyRefresh      Key = "r"
	KeyPgUp         Key = "pgup"
	KeyPgDown       Key = "pgdown"
	KeyHome         Key = "home"
	KeyEnd          Key = "end"
	KeyKillSession  Key = "x"
	KeyApprove      Key = "a"
	KeyYes          Key = "y"
	KeyNo           Key = "n"
	KeyEscape       Key = "esc"
	KeyCtrlU        Key = "ctrl+u"
	KeyCtrlD        Key = "ctrl+d"
	KeyG            Key = "g"
	KeyShiftG       Key = "G"
	KeySpawn        Key = "s"
	KeyFresh        Key = "f"
	KeyCancel       Key = "c"
	KeyEdit         Key = "e"
	KeyDeleteOrphan Key = "D"
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
func helpText(scrollPercent int, hasActiveSession, canSpawn, embedded bool) string {
	var quit string
	if embedded {
		quit = "[q/esc] back"
	} else {
		quit = "[q]uit"
	}

	scroll := "[j/k/gg/G] scroll  [ctrl+u/d] page"

	actions := "[r]efresh  [e]dit  [ga] architect"

	if hasActiveSession {
		actions = "[r]efresh  [e]dit  [x] kill  [a]pprove  [ga] architect"
	} else if canSpawn {
		actions = "[r]efresh  [e]dit  [s]pawn  [ga] architect"
	}

	return scroll + "  " + actions + "  " + quit + "  " + percentStr(scrollPercent)
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
