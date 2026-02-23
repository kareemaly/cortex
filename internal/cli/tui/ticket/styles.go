package ticket

import "github.com/charmbracelet/lipgloss"

// Status colors (reused from kanban).
var (
	backlogColor  = lipgloss.Color("245") // gray
	progressColor = lipgloss.Color("214") // yellow/orange
	doneColor     = lipgloss.Color("82")  // green
)

// General colors.
var (
	mutedColor   = lipgloss.Color("240") // muted gray
	errorColor   = lipgloss.Color("196") // red
	warningColor = lipgloss.Color("214") // yellow/orange
)

// Due date styles.
var (
	dueSoonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")) // yellow/orange

	overdueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")) // red
)

// Styles for the ticket detail view.
var (
	// Header style for the title bar.
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("62")).
			Padding(1, 1)

	// Ticket ID style.
	ticketIDStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("245"))

	// Ticket title style.
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255"))

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

	// Warning style for confirmation dialogs.
	warningStyle = lipgloss.NewStyle().
			Foreground(warningColor).
			Bold(true)
)

// Attribute panel styles.
var (
	// Section headers in attributes panel (DETAILS, SESSION).
	attributeHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("255"))

	// Attribute field labels.
	attributeLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245"))

	// Attribute field values.
	attributeValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("255"))

	// Vertical divider between body and attributes.
	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("237"))
)

// statusStyle returns the appropriate style for a ticket status.
func statusStyle(status string) lipgloss.Style {
	var color lipgloss.Color
	switch status {
	case "backlog":
		color = backlogColor
	case "progress":
		color = progressColor
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

// typeBadgePalette is a fixed palette of visually distinct ANSI 256-colors
// used for hash-based ticket type badge coloring.
var typeBadgePalette = []string{"196", "39", "35", "214", "141", "208", "49", "220"}

// typeBadgeStyle returns the appropriate style for a ticket type badge.
// "work" gets no special styling; other types get a hash-based color.
func typeBadgeStyle(ticketType string) lipgloss.Style {
	if ticketType == "work" {
		return lipgloss.NewStyle()
	}
	var sum int
	for _, b := range ticketType {
		sum += int(b)
	}
	colorCode := typeBadgePalette[sum%len(typeBadgePalette)]
	return lipgloss.NewStyle().Foreground(lipgloss.Color(colorCode))
}
