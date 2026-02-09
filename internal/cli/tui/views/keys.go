package views

import tea "github.com/charmbracelet/bubbletea"

// View-switching key constants.
const (
	keyTab      = "tab"
	keyShiftTab = "shift+tab"
	keyLBrace   = "["
	keyRBrace   = "]"
)

// isViewSwitchKey returns true if the key message is a view-switching key.
func isViewSwitchKey(msg tea.KeyMsg) bool {
	k := msg.String()
	return k == keyTab || k == keyShiftTab || k == keyLBrace || k == keyRBrace
}

// isNextView returns true if the key switches to the next view.
func isNextView(msg tea.KeyMsg) bool {
	k := msg.String()
	return k == keyTab || k == keyRBrace
}
