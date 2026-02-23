package sessions

import tea "github.com/charmbracelet/bubbletea"

type Key string

const (
	KeyQuit   Key = "q"
	KeyUp     Key = "up"
	KeyDown   Key = "down"
	KeyK      Key = "k"
	KeyJ      Key = "j"
	KeyEnter  Key = "enter"
	KeyEsc    Key = "esc"
	KeyCtrlC  Key = "ctrl+c"
	KeyCtrlU  Key = "ctrl+u"
	KeyCtrlD  Key = "ctrl+d"
	KeyG      Key = "g"
	KeyShiftG Key = "G"
	KeyR      Key = "r"
	KeyBang   Key = "!"
)

func isKey(msg tea.KeyMsg, keys ...Key) bool {
	for _, k := range keys {
		if msg.String() == string(k) {
			return true
		}
	}
	return false
}

func listHelpText() string {
	return "[j/k/gg/G] navigate  [enter] view  [!] logs  [q]uit"
}

func detailHelpText() string {
	return "[j/k/ctrl+u/d] scroll  [esc] back  [!] logs  [q]uit"
}
