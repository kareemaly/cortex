package detail

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

type Tab struct {
	Label   string
	Content string
}

type Model struct {
	title    string
	subtitle string
	tabs     []Tab
	offsets  []int
	active   int

	viewport   viewport.Model
	mdRenderer *glamour.TermRenderer
	pendingG   bool

	width, height int
	ready         bool
}

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))

	subtitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("62")).
			Padding(0, 1)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")).
				Padding(0, 1)

	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	emptyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

func New(title, subtitle string, tabs []Tab) Model {
	if len(tabs) == 0 {
		tabs = []Tab{{Label: "Overview", Content: "_No content available._"}}
	}

	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)

	return Model{
		title:      title,
		subtitle:   subtitle,
		tabs:       tabs,
		offsets:    make([]int, len(tabs)),
		mdRenderer: renderer,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.updateRendererWidth()
		m.syncViewportSize()
		m.renderActiveTab()
		return m, nil

	case tea.KeyMsg:
		m.syncViewportSize()

		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "tab", "l":
			m.switchTab(1)
			return m, nil
		case "shift+tab", "h":
			m.switchTab(-1)
			return m, nil
		case "G":
			m.pendingG = false
			m.scrollToBottom()
			m.offsets[m.active] = m.viewport.YOffset
			return m, nil
		case "g":
			if m.pendingG {
				m.pendingG = false
				m.viewport.GotoTop()
				m.offsets[m.active] = m.viewport.YOffset
			} else {
				m.pendingG = true
			}
			return m, nil
		}

		m.pendingG = false

		switch msg.String() {
		case "j", "down":
			m.viewport.ScrollDown(1)
		case "k", "up":
			m.viewport.ScrollUp(1)
		case "ctrl+d":
			m.viewport.HalfPageDown()
		case "ctrl+u":
			m.viewport.HalfPageUp()
		}

		m.offsets[m.active] = m.viewport.YOffset
		return m, nil
	}

	return m, nil
}

func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	m.syncViewportSize()

	var b strings.Builder
	b.WriteString(titleStyle.Render(m.title))
	b.WriteString("\n")
	if m.subtitle != "" {
		b.WriteString(subtitleStyle.Render(m.subtitle))
		b.WriteString("\n")
	}
	b.WriteString(m.renderTabBar())
	b.WriteString("\n")
	b.WriteString(m.viewport.View())
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("tab/h/l tabs  j/k scroll  ctrl+d/u page  gg/G jump  q close"))

	return b.String()
}

func (m *Model) updateRendererWidth() {
	wrapWidth := max(m.width-6, 40)
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(wrapWidth),
	)
	m.mdRenderer = renderer
}

func (m *Model) renderActiveTab() {
	if len(m.tabs) == 0 {
		m.viewport.SetContent(emptyStyle.Render("No content available"))
		m.viewport.SetYOffset(0)
		return
	}

	content := strings.TrimSpace(m.tabs[m.active].Content)
	if content == "" {
		content = "_No content available._"
	}

	rendered := content
	if m.mdRenderer != nil {
		if out, err := m.mdRenderer.Render(content); err == nil {
			rendered = strings.TrimSpace(out)
		}
	}

	m.viewport.SetContent(rendered)
	m.clampYOffset(m.offsets[m.active])
}

func (m *Model) switchTab(delta int) {
	if len(m.tabs) == 0 {
		return
	}

	m.offsets[m.active] = m.viewport.YOffset
	m.active = (m.active + delta + len(m.tabs)) % len(m.tabs)
	m.pendingG = false
	m.renderActiveTab()
}

func (m *Model) syncViewportSize() {
	if !m.ready {
		return
	}
	m.viewport.Width = max(m.width-2, 20)
	m.viewport.Height = max(m.height-m.headerLineCount(), 3)
	m.clampYOffset(m.viewport.YOffset)
}

func (m Model) headerLineCount() int {
	if m.subtitle == "" {
		return 3
	}
	return 4
}

func (m *Model) clampYOffset(offset int) {
	maxOffset := max(m.viewport.TotalLineCount()-m.viewport.Height, 0)
	if offset < 0 {
		offset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	m.viewport.SetYOffset(offset)
}

func (m *Model) scrollToBottom() {
	m.clampYOffset(m.viewport.TotalLineCount() - m.viewport.Height)
}

func (m Model) renderTabBar() string {
	parts := make([]string, 0, len(m.tabs))
	for i, tab := range m.tabs {
		label := fmt.Sprintf(" %s ", tab.Label)
		if i == m.active {
			parts = append(parts, activeTabStyle.Render(label))
			continue
		}
		parts = append(parts, inactiveTabStyle.Render(label))
	}
	return strings.Join(parts, "")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
