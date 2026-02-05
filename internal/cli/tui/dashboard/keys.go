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
	KeyExclaim Key = "!"
	KeyUnlink  Key = "u"
	KeyYes     Key = "y"
	KeyNo      Key = "n"
	KeyEscape  Key = "esc"
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
	return "[enter/f] focus  [s]pawn architect  [u]nlink  [r]efresh  [j/k/gg/G] navigate  [!] logs  [q]uit"
}
