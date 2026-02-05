package kanban

import "github.com/charmbracelet/lipgloss"

// Colors for the kanban board.
var (
	backlogColor  = lipgloss.Color("245") // gray
	progressColor = lipgloss.Color("214") // yellow/orange
	reviewColor   = lipgloss.Color("39")  // blue
	doneColor     = lipgloss.Color("82")  // green
	activeColor   = lipgloss.Color("212") // pink/magenta for active selection
	errorColor    = lipgloss.Color("196") // red for errors
	mutedColor    = lipgloss.Color("240") // muted gray
)

// Styles for the kanban board.
var (
	// Header style for the title bar.
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("62")).
			Padding(0, 1)

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

	reviewHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(reviewColor).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(reviewColor)

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

	// Type badge styles for ticket types.
	debugTypeBadgeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")) // red

	researchTypeBadgeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")) // blue

	choreTypeBadgeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")) // gray

	// Due date styles
	dueSoonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")) // yellow/orange

	overdueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")) // red
)

// columnHeaderStyle returns the appropriate header style for a status.
func columnHeaderStyle(status string) lipgloss.Style {
	switch status {
	case "backlog":
		return backlogHeaderStyle
	case "progress":
		return progressHeaderStyle
	case "review":
		return reviewHeaderStyle
	case "done":
		return doneHeaderStyle
	default:
		return backlogHeaderStyle
	}
}

// typeBadgeStyle returns the appropriate style for a ticket type badge.
func typeBadgeStyle(ticketType string) lipgloss.Style {
	switch ticketType {
	case "debug":
		return debugTypeBadgeStyle
	case "research":
		return researchTypeBadgeStyle
	case "chore":
		return choreTypeBadgeStyle
	default:
		return lipgloss.NewStyle()
	}
}
