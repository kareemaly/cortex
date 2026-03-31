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

	filterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
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
	filter   string
}

// New creates a new variant selector.
func New(title string, variants []string) Model {
	return Model{
		title:    title,
		variants: variants,
		cursor:   0,
	}
}

// filtered returns the variants that match the current filter (case-insensitive).
func (m Model) filtered() []string {
	if m.filter == "" {
		return m.variants
	}
	q := strings.ToLower(m.filter)
	var out []string
	for _, v := range m.variants {
		if strings.Contains(strings.ToLower(v), q) {
			out = append(out, v)
		}
	}
	return out
}

// clampCursor ensures the cursor stays within bounds of the filtered list.
func (m *Model) clampCursor(n int) {
	if n <= 0 {
		m.cursor = 0
		return
	}
	if m.cursor >= n {
		m.cursor = n - 1
	}
}

// Update handles keyboard input for the selector.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		filtered := m.filtered()
		n := len(filtered)

		switch msg.String() {
		case "j", "down":
			if n > 0 {
				m.cursor = (m.cursor + 1) % n
			}
		case "k", "up":
			if n > 0 {
				m.cursor = (m.cursor - 1 + n) % n
			}
		case "enter", " ":
			if n > 0 {
				return m, func() tea.Msg { return SelectedMsg{Name: filtered[m.cursor]} }
			}
		case "esc", "q":
			return m, func() tea.Msg { return CancelledMsg{} }
		case "backspace", "ctrl+h":
			if len(m.filter) > 0 {
				m.filter = m.filter[:len([]rune(m.filter))-1]
				m.clampCursor(len(m.filtered()))
			}
		default:
			// Append printable runes to filter.
			if len(msg.Runes) > 0 {
				m.filter += string(msg.Runes)
				m.cursor = 0
			}
		}
	}
	return m, nil
}

// View renders the selector as a popup box.
func (m Model) View() string {
	filtered := m.filtered()

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

	if m.filter != "" {
		sb.WriteString(filterStyle.Render(fmt.Sprintf("Filter: %s▌", m.filter)))
		sb.WriteString("\n")
	}

	if len(filtered) == 0 {
		sb.WriteString(normalItemStyle.Render("(no matches)"))
		sb.WriteString("\n")
	} else {
		for i, name := range filtered {
			if i == m.cursor {
				sb.WriteString(selectedItemStyle.Render(fmt.Sprintf("▶ %s", name)))
			} else {
				sb.WriteString(normalItemStyle.Render(fmt.Sprintf("  %s", name)))
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString(helpStyle.Render("type to filter  ↑/k ↓/j  enter select  esc cancel"))

	return selectorBorderStyle.Render(sb.String())
}
