package ticket

import "github.com/charmbracelet/lipgloss"

// Comment row layout constants.
const (
	CommentRowLines   = 4 // header line + 3 preview lines
	CommentRowPadding = 1 // blank line between rows
)

// Status colors (reused from kanban).
var (
	backlogColor  = lipgloss.Color("245") // gray
	progressColor = lipgloss.Color("214") // yellow/orange
	reviewColor   = lipgloss.Color("39")  // blue
	doneColor     = lipgloss.Color("82")  // green
)

// Comment type colors.
var (
	reviewRequestedColor = lipgloss.Color("214") // yellow
	doneTypeColor        = lipgloss.Color("82")  // green
	blockerColor         = lipgloss.Color("196") // red
	commentColor         = lipgloss.Color("245") // gray
)

// General colors.
var (
	mutedColor   = lipgloss.Color("240") // muted gray
	errorColor   = lipgloss.Color("196") // red
	warningColor = lipgloss.Color("214") // yellow/orange
	focusColor   = lipgloss.Color("62")  // purple (matches header)
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

// Row layout styles.
var (
	// Row 2 (comment list) with focus border.
	row2FocusedStyle = lipgloss.NewStyle().
				BorderLeft(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(focusColor)

	// Row 2 without focus.
	row2Style = lipgloss.NewStyle().
			PaddingLeft(1)

	// Comment selected (cursor highlight) in comment list.
	commentSelectedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("237"))

	// Thin horizontal line between rows.
	rowSeparatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("237"))

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

// Modal styles.
var (
	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2)

	modalHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("255"))

	modalSeparatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("237"))

	modalHelpStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	modalRepoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
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

// Type badge styles for ticket types (matching kanban).
var (
	debugTypeBadgeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")) // red

	researchTypeBadgeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")) // blue

	choreTypeBadgeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")) // gray
)

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

// commentTypeStyle returns the appropriate style for a comment type.
func commentTypeStyle(commentType string) lipgloss.Style {
	var color lipgloss.Color
	switch commentType {
	case "review_requested":
		color = reviewRequestedColor
	case "done":
		color = doneTypeColor
	case "blocker":
		color = blockerColor
	default:
		color = commentColor
	}
	return lipgloss.NewStyle().
		Foreground(color)
}
