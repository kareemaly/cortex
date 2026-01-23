package kanban

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kareemaly/cortex/internal/cli/sdk"
)

// Model is the main Bubbletea model for the kanban board.
type Model struct {
	columns      [4]Column
	client       *sdk.Client
	activeColumn int
	width        int
	height       int
	ready        bool
	err          error
	statusMsg    string
	loading      bool
}

// Message types for async operations.

// TicketsLoadedMsg is sent when tickets are successfully fetched.
type TicketsLoadedMsg struct {
	Response *sdk.ListAllTicketsResponse
}

// TicketsErrorMsg is sent when fetching tickets fails.
type TicketsErrorMsg struct {
	Err error
}

// SessionSpawnedMsg is sent when a session is successfully spawned.
type SessionSpawnedMsg struct {
	Session *sdk.SessionResponse
	Ticket  *sdk.TicketSummary
}

// SessionErrorMsg is sent when spawning a session fails.
type SessionErrorMsg struct {
	Err error
}

// ClearStatusMsg is sent to clear the status message after a delay.
type ClearStatusMsg struct{}

// New creates a new kanban model with the given client.
func New(client *sdk.Client) Model {
	return Model{
		columns: [4]Column{
			NewColumn("Backlog", "backlog"),
			NewColumn("Progress", "progress"),
			NewColumn("Review", "review"),
			NewColumn("Done", "done"),
		},
		client:  client,
		loading: true,
	}
}

// Init initializes the model and starts loading tickets.
func (m Model) Init() tea.Cmd {
	return m.loadTickets()
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case TicketsLoadedMsg:
		m.loading = false
		m.err = nil
		m.columns[0].SetTickets(msg.Response.Backlog)
		m.columns[1].SetTickets(msg.Response.Progress)
		m.columns[2].SetTickets(msg.Response.Review)
		m.columns[3].SetTickets(msg.Response.Done)
		return m, nil

	case TicketsErrorMsg:
		m.loading = false
		m.err = msg.Err
		return m, nil

	case SessionSpawnedMsg:
		m.statusMsg = fmt.Sprintf("Session spawned for: %s", msg.Ticket.Title)
		return m, tea.Batch(m.loadTickets(), m.clearStatusAfterDelay())

	case SessionErrorMsg:
		m.statusMsg = fmt.Sprintf("Error: %s", msg.Err)
		return m, m.clearStatusAfterDelay()

	case ClearStatusMsg:
		m.statusMsg = ""
		return m, nil
	}

	return m, nil
}

// handleKeyMsg handles keyboard input.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Quit.
	if isKey(msg, KeyQuit, KeyCtrlC) {
		return m, tea.Quit
	}

	// Don't process other keys while loading or if there's an error.
	if m.loading {
		return m, nil
	}

	// If there's an error, only allow refresh.
	if m.err != nil {
		if isKey(msg, KeyRefresh) {
			m.loading = true
			m.err = nil
			return m, m.loadTickets()
		}
		return m, nil
	}

	// Navigation within column.
	if isKey(msg, KeyUp, KeyK) {
		m.columns[m.activeColumn].MoveUp()
		return m, nil
	}
	if isKey(msg, KeyDown, KeyJ) {
		m.columns[m.activeColumn].MoveDown()
		return m, nil
	}

	// Navigation between columns.
	if isKey(msg, KeyLeft, KeyH) {
		if m.activeColumn > 0 {
			m.activeColumn--
		}
		return m, nil
	}
	if isKey(msg, KeyRight, KeyL) {
		if m.activeColumn < 3 {
			m.activeColumn++
		}
		return m, nil
	}

	// Spawn session.
	if isKey(msg, KeySpawn) {
		ticket := m.columns[m.activeColumn].SelectedTicket()
		if ticket != nil {
			m.statusMsg = fmt.Sprintf("Spawning session for: %s...", ticket.Title)
			return m, m.spawnSession(ticket)
		}
		return m, nil
	}

	// Architect mode.
	if isKey(msg, KeyArchitect) {
		m.statusMsg = "Opening architect session..."
		return m, m.openArchitect()
	}

	// Refresh.
	if isKey(msg, KeyRefresh) {
		m.loading = true
		return m, m.loadTickets()
	}

	return m, nil
}

// View renders the kanban board.
func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	var b strings.Builder

	// Header.
	headerLeft := headerStyle.Render("cortex1")
	headerRight := headerStyle.Render("[a]rchitect")
	headerPadding := max(m.width-lipgloss.Width(headerLeft)-lipgloss.Width(headerRight), 0)
	header := headerLeft + strings.Repeat(" ", headerPadding) + headerRight
	b.WriteString(header)
	b.WriteString("\n\n")

	// Handle error state.
	if m.err != nil {
		errMsg := errorStatusStyle.Render(fmt.Sprintf("Error: %s", m.err))
		b.WriteString(errMsg)
		b.WriteString("\n\n")
		b.WriteString("Press [r] to retry or [q] to quit\n")
		if strings.Contains(m.err.Error(), "connect") {
			b.WriteString("\nIs the daemon running? Start it with: cortexd start\n")
		}
		return b.String()
	}

	// Handle loading state.
	if m.loading {
		b.WriteString(loadingStyle.Render("Loading tickets..."))
		return b.String()
	}

	// Calculate column width.
	columnWidth := max((m.width-8)/4, 20) // -8 for padding/borders

	// Calculate available height for columns.
	// Header (1) + newlines (2) + status bar (1) + help bar (1) + margins (2) = ~7 lines overhead
	columnHeight := max(m.height-7, 5)

	// Render columns side by side.
	cols := make([]string, 4)
	for i := range m.columns {
		cols[i] = m.columns[i].View(columnWidth, i == m.activeColumn, columnHeight)
	}
	columnsView := lipgloss.JoinHorizontal(lipgloss.Top, cols...)
	b.WriteString(columnsView)
	b.WriteString("\n")

	// Status bar.
	if m.statusMsg != "" {
		b.WriteString(statusBarStyle.Render(m.statusMsg))
		b.WriteString("\n")
	} else {
		b.WriteString("\n")
	}

	// Help bar.
	b.WriteString(helpBarStyle.Render(helpText()))

	return b.String()
}

// loadTickets returns a command to load all tickets.
func (m Model) loadTickets() tea.Cmd {
	return func() tea.Msg {
		resp, err := m.client.ListAllTickets("")
		if err != nil {
			return TicketsErrorMsg{Err: err}
		}
		return TicketsLoadedMsg{Response: resp}
	}
}

// spawnSession returns a command to spawn a session for a ticket.
func (m Model) spawnSession(ticket *sdk.TicketSummary) tea.Cmd {
	return func() tea.Msg {
		session, err := m.client.SpawnSession(ticket.Status, ticket.ID, "")
		if err != nil {
			return SessionErrorMsg{Err: err}
		}
		return SessionSpawnedMsg{Session: session, Ticket: ticket}
	}
}

// openArchitect returns a command to open an architect session.
func (m Model) openArchitect() tea.Cmd {
	return func() tea.Msg {
		resp, err := m.client.SpawnArchitect("")
		if err != nil {
			return SessionErrorMsg{Err: err}
		}
		return SessionSpawnedMsg{
			Session: &sdk.SessionResponse{
				ID:         resp.Session.ID,
				TmuxWindow: resp.Session.TmuxWindow,
				StartedAt:  resp.Session.StartedAt,
			},
			Ticket: nil,
		}
	}
}

// clearStatusAfterDelay returns a command to clear the status message after a delay.
func (m Model) clearStatusAfterDelay() tea.Cmd {
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return ClearStatusMsg{}
	})
}
