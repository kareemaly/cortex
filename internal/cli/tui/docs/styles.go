package docs

import "github.com/charmbracelet/lipgloss"

// Colors matching the kanban palette plus docs-specific additions.
var (
	errorColor  = lipgloss.Color("196") // red
	mutedColor  = lipgloss.Color("240") // muted gray
	accentColor = lipgloss.Color("62")  // purple/blue (header, selected)
)

// Category colors — cycle through for different categories.
var categoryColors = []lipgloss.Color{
	lipgloss.Color("39"),  // blue
	lipgloss.Color("214"), // yellow/orange
	lipgloss.Color("82"),  // green
	lipgloss.Color("212"), // pink
	lipgloss.Color("35"),  // cyan/teal
	lipgloss.Color("245"), // gray
	lipgloss.Color("196"), // red
	lipgloss.Color("99"),  // purple
}

// categoryColor returns a consistent color for a given category index.
func categoryColor(idx int) lipgloss.Color {
	return categoryColors[idx%len(categoryColors)]
}

// Styles for the docs browser.
var (
	// Explorer pane styles.
	categoryStyle = lipgloss.NewStyle().
			Bold(true)

	docTitleStyle = lipgloss.NewStyle()

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255")).
			Background(accentColor)

	treeConnector = lipgloss.NewStyle().
			Foreground(mutedColor)

	// Preview pane styles.
	emptyPreviewStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Italic(true).
				Padding(1, 2)

	// Attribute bar styles.
	categoryBadgeStyle = lipgloss.NewStyle().
				Bold(true).
				Padding(0, 1)

	tagPillStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("240")).
			Padding(0, 1)

	dateStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	refStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("33")) // blue

	attrSeparator = lipgloss.NewStyle().
			Foreground(mutedColor)

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

	// Warn badge style for log badge.
	warnBadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	// Focus indicator for active pane.
	activePaneHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("255")).
				Background(accentColor).
				Padding(0, 1)
)
