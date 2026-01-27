package dashboard

import "github.com/charmbracelet/lipgloss"

// Colors for the dashboard.
var (
	progressColor = lipgloss.Color("214") // yellow/orange
	reviewColor   = lipgloss.Color("39")  // blue
	activeColor   = lipgloss.Color("212") // pink/magenta for active selection
	errorColor    = lipgloss.Color("196") // red for errors
	mutedColor    = lipgloss.Color("240") // muted gray
)

// Styles for the dashboard.
var (
	// Header style for the title bar.
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("62")).
			Padding(0, 1)

	// Selected row style (white on blue, matching kanban).
	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("62"))

	// Project row style.
	projectStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255"))

	// Counts style (muted).
	countsStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	// Session row style.
	sessionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	// Status badge styles.
	progressBadgeStyle = lipgloss.NewStyle().
				Foreground(progressColor)

	reviewBadgeStyle = lipgloss.NewStyle().
				Foreground(reviewColor)

	// Duration style (muted).
	durationStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	// Stale project style.
	staleStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true)

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
	mutedStyleRender = lipgloss.NewStyle().
				Foreground(mutedColor)

	// Active session icon style.
	activeIconStyle = lipgloss.NewStyle().
			Foreground(activeColor)

	// Dimmed project style for projects without active architect.
	dimmedProjectStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(mutedColor)

	// Warn badge style for log badge.
	warnBadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))
)
