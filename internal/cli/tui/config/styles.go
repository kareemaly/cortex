package config

import "github.com/charmbracelet/lipgloss"

// Colors matching the kanban/docs palette.
var (
	errorColor  = lipgloss.Color("196") // red
	mutedColor  = lipgloss.Color("240") // muted gray
	accentColor = lipgloss.Color("62")  // purple/blue (header, selected)
)

// Styles for the config browser.
var (
	// Explorer pane styles.
	sectionHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("39")) // blue

	configItemStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("214")) // orange

	treeConnector = lipgloss.NewStyle().
			Foreground(mutedColor)

	// Ejection badge styles.
	defaultBadgeStyle = lipgloss.NewStyle().
				Foreground(mutedColor)

	ejectedBadgeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("82")) // green

	// Preview pane styles.
	emptyPreviewStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Italic(true).
				Padding(1, 2)

	// Status and help bars.
	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	errorStatusStyle = lipgloss.NewStyle().
				Foreground(errorColor)

	helpBarStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	loadingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)

	// Pane header styles.
	explorerHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("255")).
				Background(lipgloss.Color("238")).
				Padding(0, 1)

	previewHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("255")).
				Background(lipgloss.Color("238")).
				Padding(0, 1)

	activePaneHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("255")).
				Background(accentColor).
				Padding(0, 1)

	// Warn badge style for log badge.
	warnBadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))
)
