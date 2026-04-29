package sessions

import tea "github.com/charmbracelet/bubbletea"

type Key string

const (
	KeyQuit       Key = "q"
	KeyUp         Key = "up"
	KeyDown       Key = "down"
	KeyK          Key = "k"
	KeyJ          Key = "j"
	KeyEnter      Key = "enter"
	KeyEsc        Key = "esc"
	KeyCtrlC      Key = "ctrl+c"
	KeyCtrlU      Key = "ctrl+u"
	KeyCtrlD      Key = "ctrl+d"
	KeyG          Key = "g"
	KeyShiftG     Key = "G"
	KeyR          Key = "r"
	KeyBang       Key = "!"
	KeyLeft       Key = "left"
	KeyRight      Key = "right"
	KeyOpenEditor Key = "o"
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
	return "←/→ dates  j/k navigate  o/↵ open  r refresh  ! logs  q quit"
}

func detailHelpText() string {
	return "j/k scroll  esc back  ! logs  q quit"
}
