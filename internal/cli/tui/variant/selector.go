package variant

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SelectedMsg is sent when the user picks a variant.
type SelectedMsg struct{ Name string }

// CancelledMsg is sent when the user cancels the selector.
type CancelledMsg struct{}

var (
	selectorBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62")).
				Padding(0, 1)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")).
				Background(lipgloss.Color("62")).
				Bold(true).
				PaddingLeft(1).
				PaddingRight(1)

	normalItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			PaddingLeft(1).
			PaddingRight(1)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255")).
			MarginBottom(1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			MarginTop(1)
)

// Model is the variant selector popup.
type Model struct {
	variants []string
	cursor   int
	title    string
}

// New creates a new variant selector.
func New(title string, variants []string) Model {
	return Model{
		title:    title,
		variants: variants,
		cursor:   0,
	}
}

// Update handles keyboard input for the selector.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.variants)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter", " ":
			if len(m.variants) > 0 {
				return m, func() tea.Msg { return SelectedMsg{Name: m.variants[m.cursor]} }
			}
		case "esc", "q":
			return m, func() tea.Msg { return CancelledMsg{} }
		}
	}
	return m, nil
}

// View renders the selector as a popup box.
func (m Model) View() string {
	if len(m.variants) == 0 {
		return selectorBorderStyle.Render(
			titleStyle.Render(m.title) + "\n" +
				normalItemStyle.Render("(no variants configured)") + "\n" +
				helpStyle.Render("esc cancel"),
		)
	}

	var sb strings.Builder
	sb.WriteString(titleStyle.Render(m.title))
	sb.WriteString("\n")

	for i, name := range m.variants {
		if i == m.cursor {
			sb.WriteString(selectedItemStyle.Render(fmt.Sprintf("▶ %s", name)))
		} else {
			sb.WriteString(normalItemStyle.Render(fmt.Sprintf("  %s", name)))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(helpStyle.Render("↑/k up  ↓/j down  enter select  esc cancel"))

	return selectorBorderStyle.Render(sb.String())
}
