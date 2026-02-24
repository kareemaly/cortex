package views

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/cli/tui/config"
	"github.com/kareemaly/cortex/internal/cli/tui/kanban"
	"github.com/kareemaly/cortex/internal/cli/tui/sessions"
	"github.com/kareemaly/cortex/internal/cli/tui/tuilog"
)

type viewID int

const (
	viewKanban viewID = iota
	viewSessions
	viewConfig
	viewCount
)

// Model is the top-level wrapper that hosts kanban, sessions, and config views.
type Model struct {
	kanban        kanban.Model
	sessions      sessions.Model
	config        config.Model
	active        viewID
	width, height int
	ready         bool
	projectName   string
}

// Tab bar styles.
var (
	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("62")).
			Padding(0, 1)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")).
				Padding(0, 1)

	tabBarStyle = lipgloss.NewStyle()
)

// New creates a new views wrapper.
func New(client *sdk.Client, logBuf *tuilog.Buffer, projectName string) Model {
	return Model{
		kanban:      kanban.New(client, logBuf),
		sessions:    sessions.New(client, logBuf),
		config:      config.New(client, logBuf),
		active:      viewKanban,
		projectName: projectName,
	}
}

// Init initializes all child models.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.kanban.Init(), m.sessions.Init(), m.config.Init())
}

// Update routes messages to child models.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// All children get the window size, minus 2 lines (tab bar + blank spacer).
		childSize := tea.WindowSizeMsg{
			Width:  msg.Width,
			Height: msg.Height - 2,
		}

		var cmd1, cmd2, cmd3 tea.Cmd
		var kanbanModel tea.Model
		kanbanModel, cmd1 = m.kanban.Update(childSize)
		m.kanban = kanbanModel.(kanban.Model)

		var sessionsModel tea.Model
		sessionsModel, cmd2 = m.sessions.Update(childSize)
		m.sessions = sessionsModel.(sessions.Model)

		var configModel tea.Model
		configModel, cmd3 = m.config.Update(childSize)
		m.config = configModel.(config.Model)

		return m, tea.Batch(cmd1, cmd2, cmd3)

	case tea.KeyMsg:
		// Check for view-switching keys first (suppressed when child captures input).
		if isViewSwitchKey(msg) && !m.isChildCapturingInput() {
			if isNextView(msg) {
				m.active = (m.active + 1) % viewCount
			} else {
				m.active = (m.active - 1 + viewCount) % viewCount
			}
			return m, nil
		}

		// Route key to active child only.
		return m.updateActiveChild(msg)

	default:
		var cmd1, cmd2, cmd3 tea.Cmd
		var kanbanModel tea.Model
		kanbanModel, cmd1 = m.kanban.Update(msg)
		m.kanban = kanbanModel.(kanban.Model)

		var sessionsModel tea.Model
		sessionsModel, cmd2 = m.sessions.Update(msg)
		m.sessions = sessionsModel.(sessions.Model)

		var configModel tea.Model
		configModel, cmd3 = m.config.Update(msg)
		m.config = configModel.(config.Model)

		return m, tea.Batch(cmd1, cmd2, cmd3)
	}
}

// updateActiveChild routes a message to the active child only.
func (m Model) updateActiveChild(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.active {
	case viewKanban:
		var cmd tea.Cmd
		var model tea.Model
		model, cmd = m.kanban.Update(msg)
		m.kanban = model.(kanban.Model)
		return m, cmd
	case viewSessions:
		var cmd tea.Cmd
		var model tea.Model
		model, cmd = m.sessions.Update(msg)
		m.sessions = model.(sessions.Model)
		return m, cmd
	case viewConfig:
		var cmd tea.Cmd
		var model tea.Model
		model, cmd = m.config.Update(msg)
		m.config = model.(config.Model)
		return m, cmd
	}
	return m, nil
}

// View renders the tab bar and the active child view.
func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	var b strings.Builder

	// Tab bar + blank margin line below it.
	b.WriteString(m.renderTabBar())
	b.WriteString("\n\n")

	// Active child view.
	switch m.active {
	case viewKanban:
		b.WriteString(m.kanban.View())
	case viewSessions:
		b.WriteString(m.sessions.View())
	case viewConfig:
		b.WriteString(m.config.View())
	}

	return b.String()
}

// renderTabBar renders the 1-line tab bar at the top.
func (m Model) renderTabBar() string {
	tabs := []struct {
		id   viewID
		name string
	}{
		{viewKanban, "Kanban"},
		{viewSessions, "Sessions"},
		{viewConfig, "Config"},
	}

	var parts []string
	for _, tab := range tabs {
		if tab.id == m.active {
			parts = append(parts, activeTabStyle.Render(tab.name))
		} else {
			parts = append(parts, inactiveTabStyle.Render(tab.name))
		}
	}

	bar := tabBarStyle.Render(strings.Join(parts, ""))
	nameStr := inactiveTabStyle.Render(m.projectName)
	padding := max(m.width-lipgloss.Width(bar)-lipgloss.Width(nameStr), 0)
	return bar + strings.Repeat(" ", padding) + nameStr
}

// isChildCapturingInput returns true when the active child is capturing keyboard input
// (e.g., text input or modal), so tab-switching keys should be suppressed.
func (m Model) isChildCapturingInput() bool {
	if m.active == viewSessions {
		return m.sessions.InputActive()
	}
	return false
}
