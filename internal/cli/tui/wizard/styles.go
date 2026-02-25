package wizard

import "github.com/charmbracelet/lipgloss"

var (
	// Palette — consistent with the rest of the TUI.
	accentColor  = lipgloss.Color("62")  // purple/blue
	mutedColor   = lipgloss.Color("240") // dim gray
	errorColor   = lipgloss.Color("196") // red
	successColor = lipgloss.Color("82")  // green
	whiteColor   = lipgloss.Color("255") // bright white

	// Sidebar header.
	sidebarHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(whiteColor).
				Background(accentColor).
				Padding(0, 1)

	// Sidebar step states.
	stepDoneStyle = lipgloss.NewStyle().
			Foreground(successColor)

	stepActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(whiteColor)

	stepPendingStyle = lipgloss.NewStyle().
				Foreground(mutedColor)

	stepErrorStyle = lipgloss.NewStyle().
			Foreground(errorColor)

	// Main pane header.
	mainHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(whiteColor).
			Background(accentColor).
			Padding(0, 1)

	// Prompt label.
	promptLabelStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(whiteColor)

	// Selected item in a list.
	selectedItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(whiteColor).
				Background(accentColor)

	// Normal list item.
	normalItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("254"))

	// Hint/help text.
	hintStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	errorStatusStyle = lipgloss.NewStyle().
				Foreground(errorColor)

	// Spinner.
	spinnerStyle = lipgloss.NewStyle().
			Foreground(accentColor)

	// Agent stream styles.
	streamTitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(whiteColor)
	streamToolStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("75"))
	streamDoneStyle     = lipgloss.NewStyle().Foreground(successColor)
	streamErrStyle      = lipgloss.NewStyle().Foreground(errorColor)
	streamContentStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	streamSubagentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("213"))
	streamReasonStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("180"))
	streamStatsStyle    = lipgloss.NewStyle().Foreground(mutedColor)

	// Result section styles.
	resultHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(whiteColor)
	resultCheckStyle  = lipgloss.NewStyle().Foreground(successColor)
	resultCrossStyle  = lipgloss.NewStyle().Foreground(errorColor)
	resultBulletStyle = lipgloss.NewStyle().Foreground(mutedColor)

	// Confirm highlight.
	confirmYesStyle = lipgloss.NewStyle().Bold(true).Foreground(successColor)
	confirmNoStyle  = lipgloss.NewStyle().Bold(true).Foreground(errorColor)

	// Divider.
	dividerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
)
