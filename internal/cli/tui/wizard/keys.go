package wizard

import tea "github.com/charmbracelet/bubbletea"

// Key represents a keyboard key.
type Key string

const (
	KeyCtrlC Key = "ctrl+c"
	KeyEnter Key = "enter"
	KeyUp    Key = "up"
	KeyDown  Key = "down"
	KeyK     Key = "k"
	KeyJ     Key = "j"
	KeyY     Key = "y"
	KeyN     Key = "n"
)

func isKey(msg tea.KeyMsg, keys ...Key) bool {
	for _, k := range keys {
		if msg.String() == string(k) {
			return true
		}
	}
	return false
}
