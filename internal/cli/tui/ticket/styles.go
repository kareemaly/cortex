package ticket

import "github.com/charmbracelet/lipgloss"

// Status colors (reused from kanban).
var (
	backlogColor  = lipgloss.Color("245") // gray
	progressColor = lipgloss.Color("214") // yellow/orange
	reviewColor   = lipgloss.Color("39")  // blue
	doneColor     = lipgloss.Color("82")  // green
)

// Comment type colors.
var (
	decisionColor     = lipgloss.Color("39")  // blue
	scopeChangeColor  = lipgloss.Color("214") // yellow
	blockerColor      = lipgloss.Color("196") // red
	progressTypeColor = lipgloss.Color("82")  // green
	questionColor     = lipgloss.Color("87")  // cyan
	rejectionColor    = lipgloss.Color("197") // magenta
	generalColor      = lipgloss.Color("245") // gray
)

// General colors.
var (
	mutedColor = lipgloss.Color("240") // muted gray
	errorColor = lipgloss.Color("196") // red
)

// Styles for the ticket detail view.
var (
	// Header style for the title bar.
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("62")).
			Padding(0, 1)

	// Ticket ID style.
	ticketIDStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("245"))

	// Ticket title style.
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255"))

	// Section header style.
	sectionHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("245"))

	// Label style for field names.
	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	// Value style for field values.
	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255"))

	// Help bar at the bottom.
	helpBarStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	// Error status style.
	errorStatusStyle = lipgloss.NewStyle().
				Foreground(errorColor)

	// Loading indicator.
	loadingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)
)

// statusStyle returns the appropriate style for a ticket status.
func statusStyle(status string) lipgloss.Style {
	var color lipgloss.Color
	switch status {
	case "backlog":
		color = backlogColor
	case "progress":
		color = progressColor
	case "review":
		color = reviewColor
	case "done":
		color = doneColor
	default:
		color = backlogColor
	}
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("0")).
		Background(color).
		Padding(0, 1)
}

// commentTypeStyle returns the appropriate style for a comment type.
func commentTypeStyle(commentType string) lipgloss.Style {
	var color lipgloss.Color
	switch commentType {
	case "decision":
		color = decisionColor
	case "scope_change":
		color = scopeChangeColor
	case "blocker":
		color = blockerColor
	case "progress":
		color = progressTypeColor
	case "question":
		color = questionColor
	case "rejection":
		color = rejectionColor
	default:
		color = generalColor
	}
	return lipgloss.NewStyle().
		Foreground(color)
}
