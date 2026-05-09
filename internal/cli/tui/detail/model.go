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

type EditResult struct {
	Title    string
	Subtitle string
	Tabs     []Tab
}

type EditFunc = tea.Cmd

type Option func(*Model)

type Model struct {
	title    string
	subtitle string
	tabs     []Tab
	offsets  []int
	active   int

	ticketID  string
	indexPath string
	onEdit    EditFunc
	status    string

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

type editFinishedMsg struct {
	result EditResult
	err    error
}

func EditFinished(result EditResult, err error) tea.Msg {
	return editFinishedMsg{result: result, err: err}
}

func WithEditableTicket(ticketID, indexPath string, onEdit EditFunc) Option {
	return func(m *Model) {
		m.ticketID = ticketID
		m.indexPath = indexPath
		m.onEdit = onEdit
	}
}

func New(title, subtitle string, tabs []Tab, opts ...Option) Model {
	if len(tabs) == 0 {
		tabs = []Tab{{Label: "Overview", Content: "_No content available._"}}
	}

	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)

	model := Model{
		title:      title,
		subtitle:   subtitle,
		tabs:       tabs,
		offsets:    make([]int, len(tabs)),
		mdRenderer: renderer,
	}

	for _, opt := range opts {
		opt(&model)
	}

	return model
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

	case editFinishedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Edit failed: %v", msg.err)
			return m, nil
		}

		m.applyEditResult(msg.result)
		m.status = fmt.Sprintf("Reloaded ticket %s", m.ticketID)
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
		case "e":
			if m.canEdit() {
				m.pendingG = false
				m.status = ""
				return m, m.onEdit
			}
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
	if m.status != "" {
		b.WriteString(helpStyle.Render(m.status))
		b.WriteString("\n")
	}
	b.WriteString(helpStyle.Render(m.helpText()))

	return b.String()
}

func (m Model) canEdit() bool {
	return m.ticketID != "" && m.indexPath != "" && m.onEdit != nil
}

func (m *Model) applyEditResult(result EditResult) {
	currentLabel := ""
	if len(m.tabs) > 0 && m.active < len(m.tabs) {
		currentLabel = m.tabs[m.active].Label
	}

	m.title = result.Title
	m.subtitle = result.Subtitle
	m.tabs = result.Tabs
	if len(m.tabs) == 0 {
		m.tabs = []Tab{{Label: "Overview", Content: "_No content available._"}}
	}

	m.offsets = make([]int, len(m.tabs))
	m.active = 0
	for i, tab := range m.tabs {
		if tab.Label == currentLabel {
			m.active = i
			break
		}
	}

	m.renderActiveTab()
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
	lines := 3
	if m.subtitle != "" {
		lines++
	}
	if m.status != "" {
		lines++
	}
	return lines
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

func (m Model) helpText() string {
	parts := []string{"tab/h/l tabs", "j/k scroll", "ctrl+d/u page", "gg/G jump"}
	if m.canEdit() {
		parts = append(parts, "e edit")
	}
	parts = append(parts, "q close")
	return strings.Join(parts, "  ")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
