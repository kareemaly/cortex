package kanban

import "github.com/charmbracelet/lipgloss"

// Colors for the kanban board.
var (
	backlogColor  = lipgloss.Color("245") // gray
	progressColor = lipgloss.Color("214") // yellow/orange
	doneColor     = lipgloss.Color("82")  // green
	activeColor   = lipgloss.Color("212") // pink/magenta for active selection
	errorColor    = lipgloss.Color("196") // red for errors
	mutedColor    = lipgloss.Color("240") // muted gray
)

// Styles for the kanban board.
var (
	// Column header styles by status.
	backlogHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(backlogColor).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(backlogColor)

	progressHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(progressColor).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(progressColor)

	doneHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(doneColor).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(doneColor)

	// Column container styles.
	columnStyle = lipgloss.NewStyle().
			Padding(0, 1)

	activeColumnStyle = lipgloss.NewStyle().
				Padding(0, 1).
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(activeColor)

	// Ticket card styles.
	ticketStyle = lipgloss.NewStyle().
			Padding(0, 1)

	selectedTicketStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Bold(true).
				Foreground(lipgloss.Color("255")).
				Background(lipgloss.Color("62"))

	// Ticket with active session indicator.
	activeSessionStyle = lipgloss.NewStyle().
				Foreground(progressColor)

	// Ticket date style (muted).
	ticketDateStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(mutedColor)

	// Status bar at the bottom.
	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	errorStatusStyle = lipgloss.NewStyle().
				Foreground(errorColor)

	// Help bar at the very bottom.
	helpBarStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	// Loading indicator.
	loadingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)

	// Muted style for scroll indicators.
	mutedStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	// Warn badge style for log badge.
	warnBadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	// Due date styles
	dueSoonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")) // yellow/orange

	overdueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")) // red

	// Orphaned session style (warning color).
	orphanedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")) // yellow/orange
)

// selectedFgColor is the default foreground color for selected card text.
const selectedFgColor = "255"

// inlineFgColorChange returns a raw ANSI escape that changes only the foreground
// color without resetting other attributes. This preserves the outer background
// set by selectedTicketStyle.Render().
func inlineFgColorChange(colorCode string) string {
	return "\x1b[38;5;" + colorCode + "m"
}

// typeBadgePalette is a fixed palette of visually distinct ANSI 256-colors
// used for hash-based ticket type badge coloring.
var typeBadgePalette = []string{"196", "39", "35", "214", "141", "208", "49", "220"}

// typeBadgeColorCode returns the 256-color code for a ticket type badge.
// "work" gets the default foreground; other types are hashed into a color palette.
func typeBadgeColorCode(ticketType string) string {
	if ticketType == "work" {
		return selectedFgColor
	}
	var sum int
	for _, b := range ticketType {
		sum += int(b)
	}
	return typeBadgePalette[sum%len(typeBadgePalette)]
}

// dueDateColorCode returns the 256-color code for a due date indicator.
func dueDateColorCode(overdue bool) string {
	if overdue {
		return "196"
	}
	return "214"
}

// columnHeaderStyle returns the appropriate header style for a status.
func columnHeaderStyle(status string) lipgloss.Style {
	switch status {
	case "backlog":
		return backlogHeaderStyle
	case "progress":
		return progressHeaderStyle
	case "done":
		return doneHeaderStyle
	default:
		return backlogHeaderStyle
	}
}

// typeBadgeStyle returns the appropriate style for a ticket type badge.
// "work" gets no special styling; other types get a hash-based color.
func typeBadgeStyle(ticketType string) lipgloss.Style {
	if ticketType == "work" {
		return lipgloss.NewStyle()
	}
	colorCode := typeBadgeColorCode(ticketType)
	return lipgloss.NewStyle().Foreground(lipgloss.Color(colorCode))
}
