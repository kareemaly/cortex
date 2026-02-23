package notes

import "github.com/charmbracelet/lipgloss"

var (
	errorColor  = lipgloss.Color("196") // red
	mutedColor  = lipgloss.Color("240") // muted gray
	accentColor = lipgloss.Color("62")  // purple/blue (header, selected)
)

// Styles for the notes view.
var (
	selectedNoteStyle = lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true)

	overdueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	dueSoonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	dueBadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))

	createdStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

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

	inputLabelStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

	warnBadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	deleteModalStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Bold(true)
)
