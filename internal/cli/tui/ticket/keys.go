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
	KeyH            Key = "h"
	KeyL            Key = "l"
	KeyO            Key = "o"
	KeyEnter        Key = "enter"
	KeyTab          Key = "tab"
	KeyShiftTab     Key = "shift+tab"
	KeyLeftBracket  Key = "["
	KeyRightBracket Key = "]"
	KeyDiff         Key = "d"
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
func helpText(scrollPercent int, hasActiveSession, hasReviewRequests, canSpawn, embedded bool, focusedRow int) string {
	var quit string
	if embedded {
		quit = "[q/esc] back"
	} else {
		quit = "[q]uit"
	}

	var scroll string
	if focusedRow == 1 {
		scroll = "[Tab/[/]] body  [j/k] select  [gg/G] first/last  [o/Enter] open"
	} else {
		scroll = "[Tab/[/]] comments  [j/k/gg/G] scroll  [ctrl+u/d] page"
	}

	actions := "[r]efresh  [ga] architect"

	if hasActiveSession {
		sessionActions := "[x] kill"
		if hasReviewRequests {
			sessionActions += "  [a]pprove"
		}
		actions = "[r]efresh  " + sessionActions + "  [ga] architect"
	} else if canSpawn {
		actions = "[r]efresh  [s]pawn  [ga] architect"
	}

	return scroll + "  " + actions + "  " + quit + "  " + percentStr(scrollPercent)
}

// modalHelpText returns help text for the detail modal.
func modalHelpText(isReview, hasAction bool) string {
	base := "[Esc/q] close  [j/k] scroll"
	if isReview {
		actions := "  [a]pprove  [x] reject"
		if hasAction {
			actions += "  [d]iff"
		}
		return base + actions
	}
	return base
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
