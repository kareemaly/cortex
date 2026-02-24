package sessions

import "github.com/charmbracelet/lipgloss"

var (
	errorColor  = lipgloss.Color("196")
	mutedColor  = lipgloss.Color("240")
	accentColor = lipgloss.Color("62")
)

var (
	selectedItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("255")).
				Background(accentColor)

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

	warnBadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	dateStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	ticketRefStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	detailHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("255"))

	detailMetaStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	// Date strip styles
	activeDateStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255")).
			Background(accentColor).
			Padding(0, 1)

	inactiveDateStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Padding(0, 1)

	dateStripStyle = lipgloss.NewStyle().
			PaddingBottom(1)

	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238"))

	timeStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	typeLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Width(10)
)

var (
	architectColor = lipgloss.Color("141")
	researchColor  = lipgloss.Color("39")
	workColor      = lipgloss.Color("82")
)

func typeBadgeStyle(sessionType string) lipgloss.Style {
	var color lipgloss.Color
	switch sessionType {
	case "architect":
		color = architectColor
	case "research":
		color = researchColor
	case "work":
		color = workColor
	default:
		color = mutedColor
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("0")).
		Background(color).
		Bold(true).
		Padding(0, 1)
}
