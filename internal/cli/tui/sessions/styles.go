package sessions

import "github.com/charmbracelet/lipgloss"

var (
	errorColor  = lipgloss.Color("196")
	mutedColor  = lipgloss.Color("240")
	accentColor = lipgloss.Color("62")
)

// Column widths for aligned rendering.
const (
	colTime     = 7  // "15:04" + padding
	colType     = 10 // "research" + padding
	colDuration = 8  // " 1h 23m"
	colGutter   = 2  // left gutter for cursor indicator
)

var (
	selectedItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("255")).
				Background(accentColor)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	errorStatusStyle = lipgloss.NewStyle().
				Foreground(errorColor)

	helpBarStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	loadingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)

	emptyStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true)

	warnBadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	ticketRefStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	detailHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("255"))

	detailLabelStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Width(12)

	detailValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("255"))

	// Date strip styles
	activeDateStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255")).
			Background(accentColor).
			Padding(0, 1)

	inactiveDateStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Padding(0, 1)

	dateStripStyle = lipgloss.NewStyle()

	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238"))

	timeStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	durationStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("239"))

	dateCountStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))
)

var (
	architectColor = lipgloss.Color("141")
	researchColor  = lipgloss.Color("39")
	workColor      = lipgloss.Color("82")
	collabColor    = lipgloss.Color("214") // amber/yellow
)

// typeColor returns the color for a session type.
func typeColor(sessionType string) lipgloss.Color {
	switch sessionType {
	case "architect":
		return architectColor
	case "research":
		return researchColor
	case "work":
		return workColor
	case "collab":
		return collabColor
	default:
		return mutedColor
	}
}

// typeColorCode returns the ANSI 256-color code string for a session type.
func typeColorCode(sessionType string) string {
	switch sessionType {
	case "architect":
		return "141"
	case "research":
		return "39"
	case "work":
		return "82"
	case "collab":
		return "214"
	default:
		return "240"
	}
}

func typeBadgeStyle(sessionType string) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("0")).
		Background(typeColor(sessionType)).
		Bold(true).
		Padding(0, 1)
}

// typeLabelColorStyle returns a style with the type's foreground color, fixed width.
func typeLabelColorStyle(sessionType string) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(typeColor(sessionType)).
		Width(colType)
}

// inlineFgColor returns a raw ANSI escape that changes only the foreground
// without resetting other attributes (preserves the selected background).
func inlineFgColor(colorCode string) string {
	return "\x1b[38;5;" + colorCode + "m"
}

// resetFg returns an ANSI escape that restores the default selected foreground.
func resetFg() string {
	return "\x1b[38;5;255m"
}
