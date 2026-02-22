package notes

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
	KeyN      Key = "n"
	KeyE      Key = "e"
	KeyT      Key = "t"
	KeyD      Key = "d"
	KeySpace  Key = " "
	KeyEnter  Key = "enter"
	KeyEsc    Key = "esc"
	KeyY      Key = "y"
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

// helpText returns the help bar text for the notes view.
func helpText() string {
	return "[j/k/gg/G] navigate  [n]ew  [e]dit text  [t] due date  [space] done  [d]elete  [!] logs  [q]uit"
}

// inputHelpText returns the help bar text when in input mode.
func inputHelpText() string {
	return "[enter] confirm  [esc] cancel"
}
