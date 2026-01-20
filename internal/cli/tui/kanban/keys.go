package kanban

import tea "github.com/charmbracelet/bubbletea"

// Key represents a keyboard key.
type Key string

// Key constants for navigation and actions.
const (
	KeyQuit      Key = "q"
	KeyUp        Key = "up"
	KeyDown      Key = "down"
	KeyLeft      Key = "left"
	KeyRight     Key = "right"
	KeyK         Key = "k"
	KeyJ         Key = "j"
	KeyH         Key = "h"
	KeyL         Key = "l"
	KeySpawn     Key = "s"
	KeyArchitect Key = "a"
	KeyRefresh   Key = "r"
	KeyEnter     Key = "enter"
	KeyCtrlC     Key = "ctrl+c"
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

// helpText returns the help bar text for the kanban board.
func helpText() string {
	return "[s]pawn  [a]rchitect  [r]efresh  [h/l] columns  [j/k] navigate  [q]uit"
}
