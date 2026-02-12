package config

import tea "github.com/charmbracelet/bubbletea"

// Key represents a keyboard key.
type Key string

// Key constants for navigation and actions.
const (
	KeyQuit   Key = "q"
	KeyUp     Key = "up"
	KeyDown   Key = "down"
	KeyK      Key = "k"
	KeyJ      Key = "j"
	KeyH      Key = "h"
	KeyL      Key = "l"
	KeyE      Key = "e"
	KeyC      Key = "c"
	KeyCtrlC  Key = "ctrl+c"
	KeyCtrlU  Key = "ctrl+u"
	KeyCtrlD  Key = "ctrl+d"
	KeyG      Key = "g"
	KeyShiftG Key = "G"
	KeyR      Key = "r"
	KeyBang   Key = "!"
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

// helpText returns the help bar text for the config browser.
func helpText() string {
	return "[j/k] navigate  [e]ject/edit  [c]onfig  [h/l] pane  [ctrl+u/d] scroll  [r]efresh  [!] logs  [q]uit"
}
