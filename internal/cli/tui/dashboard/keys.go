package dashboard

import tea "github.com/charmbracelet/bubbletea"

// Key represents a keyboard key.
type Key string

// Key constants for navigation and actions.
const (
	KeyQuit    Key = "q"
	KeyUp      Key = "up"
	KeyDown    Key = "down"
	KeyK       Key = "k"
	KeyJ       Key = "j"
	KeyH       Key = "h"
	KeyL       Key = "l"
	KeyEnter   Key = "enter"
	KeyFocus   Key = "f"
	KeySpawn   Key = "s"
	KeyRefresh Key = "r"
	KeyCtrlC   Key = "ctrl+c"
	KeyCtrlU   Key = "ctrl+u"
	KeyCtrlD   Key = "ctrl+d"
	KeyG       Key = "g"
	KeyShiftG  Key = "G"
)

// isKey checks if a key message matches any of the given key constants.
func isKey(msg tea.KeyMsg, keys ...Key) bool {
	for _, k := range keys {
		if msg.String() == string(k) {
			return true
		}
	}
	return false
}

// helpText returns the help bar text for the dashboard.
func helpText() string {
	return "[f]ocus  [s]pawn architect  [r]efresh  [j/k/gg/G] navigate  [enter/l] expand  [h] collapse  [q]uit"
}
