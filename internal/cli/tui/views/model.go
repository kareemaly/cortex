package views

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/cli/tui/config"
	"github.com/kareemaly/cortex/internal/cli/tui/docs"
	"github.com/kareemaly/cortex/internal/cli/tui/kanban"
	"github.com/kareemaly/cortex/internal/cli/tui/tuilog"
)

type viewID int

const (
	viewKanban viewID = iota
	viewDocs
	viewConfig
	viewCount // sentinel for wrapping
)

// Model is the top-level wrapper that hosts kanban, docs, and config views.
type Model struct {
	kanban        kanban.Model
	docs          docs.Model
	config        config.Model
	active        viewID
	width, height int
	ready         bool
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
func New(client *sdk.Client, logBuf *tuilog.Buffer) Model {
	return Model{
		kanban: kanban.New(client, logBuf),
		docs:   docs.New(client, logBuf),
		config: config.New(client, logBuf),
		active: viewKanban,
	}
}

// Init initializes all child models.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.kanban.Init(), m.docs.Init(), m.config.Init())
}

// Update routes messages to child models.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// All children get the window size, minus 1 line for tab bar.
		childSize := tea.WindowSizeMsg{
			Width:  msg.Width,
			Height: msg.Height - 1,
		}

		var cmd1, cmd2, cmd3 tea.Cmd
		var kanbanModel tea.Model
		kanbanModel, cmd1 = m.kanban.Update(childSize)
		m.kanban = kanbanModel.(kanban.Model)

		var docsModel tea.Model
		docsModel, cmd2 = m.docs.Update(childSize)
		m.docs = docsModel.(docs.Model)

		var configModel tea.Model
		configModel, cmd3 = m.config.Update(childSize)
		m.config = configModel.(config.Model)

		return m, tea.Batch(cmd1, cmd2, cmd3)

	case tea.KeyMsg:
		// Check for view-switching keys first.
		if isViewSwitchKey(msg) {
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
		// Non-key, non-size messages go to all children.
		// Each child only processes its own typed messages.
		var cmd1, cmd2, cmd3 tea.Cmd
		var kanbanModel tea.Model
		kanbanModel, cmd1 = m.kanban.Update(msg)
		m.kanban = kanbanModel.(kanban.Model)

		var docsModel tea.Model
		docsModel, cmd2 = m.docs.Update(msg)
		m.docs = docsModel.(docs.Model)

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
	case viewDocs:
		var cmd tea.Cmd
		var model tea.Model
		model, cmd = m.docs.Update(msg)
		m.docs = model.(docs.Model)
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

	// Tab bar.
	b.WriteString(m.renderTabBar())

	// Active child view.
	switch m.active {
	case viewKanban:
		b.WriteString(m.kanban.View())
	case viewDocs:
		b.WriteString(m.docs.View())
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
		{viewDocs, "Docs"},
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
	padding := max(m.width-lipgloss.Width(bar), 0)
	return bar + strings.Repeat(" ", padding) + "\n"
}
