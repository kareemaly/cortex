package kanban

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/cli/tui/ticket"
)

// Model is the main Bubbletea model for the kanban board.
type Model struct {
	columns       [4]Column
	client        *sdk.Client
	activeColumn  int
	width         int
	height        int
	ready         bool
	err           error
	statusMsg     string
	statusIsError bool
	loading       bool

	// Modal state for orphaned session handling
	showOrphanModal bool
	orphanedTicket  *sdk.TicketSummary

	// Vim navigation state
	pendingG bool // tracking 'g' key for 'gg' sequence

	// Ticket detail view state
	showDetail  bool
	detailModel *ticket.Model

	// SSE subscription state
	eventCh      <-chan sdk.Event
	cancelEvents context.CancelFunc
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

// OrphanedSessionMsg is sent when spawn encounters an orphaned session.
type OrphanedSessionMsg struct {
	Ticket *sdk.TicketSummary
}

// FocusSuccessMsg is sent when a window is successfully focused.
type FocusSuccessMsg struct {
	Window string
}

// FocusErrorMsg is sent when focusing a window fails.
type FocusErrorMsg struct {
	Err error
}

// SessionApprovedMsg is sent when a session is successfully approved.
type SessionApprovedMsg struct {
	Ticket *sdk.TicketSummary
}

// sseConnectedMsg is sent when the SSE connection is established.
type sseConnectedMsg struct {
	ch     <-chan sdk.Event
	cancel context.CancelFunc
}

// EventMsg is sent when an SSE event is received.
type EventMsg struct{}

// ApproveErrorMsg is sent when approving a session fails.
type ApproveErrorMsg struct {
	Err error
}

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
	return tea.Batch(m.loadTickets(), m.subscribeEvents())
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle close from ticket detail view.
	if _, ok := msg.(ticket.CloseDetailMsg); ok {
		m.showDetail = false
		m.detailModel = nil
		m.loading = true
		return m, m.loadTickets()
	}

	// Delegate to detail model when active.
	if m.showDetail && m.detailModel != nil {
		if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
			m.width = sizeMsg.Width
			m.height = sizeMsg.Height
			m.ready = true
		}
		var cmd tea.Cmd
		updatedModel, cmd := m.detailModel.Update(msg)
		if dm, ok := updatedModel.(ticket.Model); ok {
			m.detailModel = &dm
		}
		return m, cmd
	}

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
		m.statusIsError = false
		return m, tea.Batch(m.loadTickets(), m.clearStatusAfterDelay())

	case SessionErrorMsg:
		m.statusMsg = fmt.Sprintf("Error: %s", msg.Err)
		m.statusIsError = true
		return m, m.clearStatusAfterDelay()

	case OrphanedSessionMsg:
		m.showOrphanModal = true
		m.orphanedTicket = msg.Ticket
		m.statusMsg = ""
		return m, nil

	case SessionApprovedMsg:
		m.statusMsg = fmt.Sprintf("Approved session for: %s", msg.Ticket.Title)
		m.statusIsError = false
		return m, tea.Batch(m.loadTickets(), m.clearStatusAfterDelay())

	case ApproveErrorMsg:
		m.statusMsg = fmt.Sprintf("Approve error: %s", msg.Err)
		m.statusIsError = true
		return m, m.clearStatusAfterDelay()

	case FocusSuccessMsg:
		m.statusMsg = fmt.Sprintf("Focused: %s", msg.Window)
		m.statusIsError = false
		return m, m.clearStatusAfterDelay()

	case FocusErrorMsg:
		m.statusMsg = fmt.Sprintf("Focus error: %s", msg.Err)
		m.statusIsError = true
		return m, m.clearStatusAfterDelay()

	case ClearStatusMsg:
		m.statusMsg = ""
		m.statusIsError = false
		return m, nil

	case sseConnectedMsg:
		m.eventCh = msg.ch
		m.cancelEvents = msg.cancel
		return m, m.waitForEvent()

	case EventMsg:
		cmds := []tea.Cmd{m.loadTickets(), m.waitForEvent()}
		if m.showDetail && m.detailModel != nil {
			cmds = append(cmds, func() tea.Msg { return ticket.RefreshMsg{} })
		}
		return m, tea.Batch(cmds...)
	}

	return m, nil
}

// handleKeyMsg handles keyboard input.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Quit.
	if isKey(msg, KeyQuit, KeyCtrlC) {
		if m.cancelEvents != nil {
			m.cancelEvents()
		}
		return m, tea.Quit
	}

	// Modal state takes priority.
	if m.showOrphanModal {
		return m.handleOrphanModalKey(msg)
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

	// Handle 'G' - jump to last
	if isKey(msg, KeyShiftG) {
		m.pendingG = false
		m.columns[m.activeColumn].JumpToLast()
		return m, nil
	}

	// Handle 'g' key for 'gg' sequence
	if isKey(msg, KeyG) {
		if m.pendingG {
			// Second 'g' - jump to first
			m.pendingG = false
			m.columns[m.activeColumn].JumpToFirst()
		} else {
			// First 'g' - set pending state
			m.pendingG = true
		}
		return m, nil
	}

	// Clear pending g on any other key
	m.pendingG = false

	// Scroll up (ctrl+u)
	if isKey(msg, KeyCtrlU) {
		m.columns[m.activeColumn].ScrollUp(10)
		return m, nil
	}

	// Scroll down (ctrl+d)
	if isKey(msg, KeyCtrlD) {
		m.columns[m.activeColumn].ScrollDown(10)
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
		t := m.columns[m.activeColumn].SelectedTicket()
		if t != nil {
			m.statusMsg = fmt.Sprintf("Spawning session for: %s...", t.Title)
			m.statusIsError = false
			return m, m.spawnSession(t)
		}
		return m, nil
	}

	// Focus tmux window.
	if isKey(msg, KeyFocus) {
		t := m.columns[m.activeColumn].SelectedTicket()
		if t != nil && t.HasActiveSession {
			m.statusMsg = "Focusing window..."
			m.statusIsError = false
			return m, m.focusTicket(t)
		}
		if t != nil && !t.HasActiveSession {
			m.statusMsg = "No active session"
			m.statusIsError = false
			return m, m.clearStatusAfterDelay()
		}
		return m, nil
	}

	// Approve session.
	if isKey(msg, KeyApprove) {
		t := m.columns[m.activeColumn].SelectedTicket()
		if t != nil && t.HasActiveSession {
			m.statusMsg = fmt.Sprintf("Approving session for: %s...", t.Title)
			m.statusIsError = false
			return m, m.approveSession(t)
		}
		if t != nil && !t.HasActiveSession {
			m.statusMsg = "No active session to approve"
			m.statusIsError = false
			return m, m.clearStatusAfterDelay()
		}
		return m, nil
	}

	// Open ticket detail.
	if isKey(msg, KeyOpen, KeyEnter) {
		t := m.columns[m.activeColumn].SelectedTicket()
		if t != nil {
			detailModel := ticket.NewEmbedded(m.client, t.ID)
			m.detailModel = &detailModel
			m.showDetail = true
			initCmd := m.detailModel.Init()
			sizeCmd := func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			}
			return m, tea.Batch(initCmd, sizeCmd)
		}
		return m, nil
	}

	// Refresh.
	if isKey(msg, KeyRefresh) {
		m.loading = true
		return m, m.loadTickets()
	}

	return m, nil
}

// handleOrphanModalKey handles keyboard input when the orphan modal is shown.
func (m Model) handleOrphanModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case isKey(msg, KeyRefresh): // 'r' for resume
		m.showOrphanModal = false
		m.statusMsg = fmt.Sprintf("Resuming session for: %s...", m.orphanedTicket.Title)
		m.statusIsError = false
		return m, m.spawnSessionWithMode(m.orphanedTicket, "resume")

	case isKey(msg, KeyFresh): // 'f' for fresh
		m.showOrphanModal = false
		m.statusMsg = fmt.Sprintf("Starting fresh session for: %s...", m.orphanedTicket.Title)
		m.statusIsError = false
		return m, m.spawnSessionWithMode(m.orphanedTicket, "fresh")

	case isKey(msg, KeyCancel, KeyEscape): // 'c' or Esc for cancel
		m.showOrphanModal = false
		m.orphanedTicket = nil
		m.statusMsg = "Spawn cancelled"
		m.statusIsError = false
		return m, m.clearStatusAfterDelay()
	}
	return m, nil
}

// View renders the kanban board.
func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	// Delegate to detail view when active.
	if m.showDetail && m.detailModel != nil {
		return m.detailModel.View()
	}

	var b strings.Builder

	// Header.
	headerLeft := headerStyle.Render("cortex1")
	headerPadding := max(m.width-lipgloss.Width(headerLeft), 0)
	header := headerLeft + strings.Repeat(" ", headerPadding)
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
	columnWidth := max((m.width-2)/4, 20) // -2 for minimal side margins

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

	// Status bar / Modal.
	if m.showOrphanModal {
		b.WriteString(m.renderOrphanModal())
	} else {
		if m.statusMsg != "" {
			style := statusBarStyle
			if m.statusIsError {
				style = errorStatusStyle
			}
			b.WriteString(style.Render(m.statusMsg))
			b.WriteString("\n")
		} else {
			b.WriteString("\n")
		}
		b.WriteString(helpBarStyle.Render(helpText()))
	}

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

// subscribeEvents returns a command that connects to the SSE event stream.
func (m Model) subscribeEvents() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		ch, err := m.client.SubscribeEvents(ctx)
		if err != nil {
			cancel()
			return nil // graceful degradation
		}
		return sseConnectedMsg{ch: ch, cancel: cancel}
	}
}

// waitForEvent returns a command that waits for the next SSE event.
func (m Model) waitForEvent() tea.Cmd {
	if m.eventCh == nil {
		return nil
	}
	ch := m.eventCh
	return func() tea.Msg {
		_, ok := <-ch
		if !ok {
			return nil
		}
		return EventMsg{}
	}
}

// spawnSession returns a command to spawn a session for a ticket.
func (m Model) spawnSession(ticket *sdk.TicketSummary) tea.Cmd {
	return func() tea.Msg {
		session, err := m.client.SpawnSession(ticket.Status, ticket.ID, "normal")
		if err != nil {
			if apiErr, ok := err.(*sdk.APIError); ok && apiErr.IsOrphanedSession() {
				return OrphanedSessionMsg{Ticket: ticket}
			}
			return SessionErrorMsg{Err: err}
		}
		return SessionSpawnedMsg{Session: session, Ticket: ticket}
	}
}

// spawnSessionWithMode returns a command to spawn a session with a specific mode.
func (m Model) spawnSessionWithMode(ticket *sdk.TicketSummary, mode string) tea.Cmd {
	return func() tea.Msg {
		session, err := m.client.SpawnSession(ticket.Status, ticket.ID, mode)
		if err != nil {
			return SessionErrorMsg{Err: err}
		}
		return SessionSpawnedMsg{Session: session, Ticket: ticket}
	}
}

// focusTicket returns a command to focus the tmux window for a ticket.
func (m Model) focusTicket(ticket *sdk.TicketSummary) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.FocusTicket(ticket.ID); err != nil {
			return FocusErrorMsg{Err: err}
		}
		return FocusSuccessMsg{Window: ticket.Title}
	}
}

// approveSession returns a command to approve a session for a ticket.
func (m Model) approveSession(ticket *sdk.TicketSummary) tea.Cmd {
	return func() tea.Msg {
		// First get the ticket to find the session ID
		fullTicket, err := m.client.GetTicket(ticket.Status, ticket.ID)
		if err != nil {
			return ApproveErrorMsg{Err: err}
		}
		if fullTicket.Session == nil {
			return ApproveErrorMsg{Err: fmt.Errorf("no session found")}
		}
		if err := m.client.ApproveSession(fullTicket.Session.ID); err != nil {
			return ApproveErrorMsg{Err: err}
		}
		return SessionApprovedMsg{Ticket: ticket}
	}
}

// clearStatusAfterDelay returns a command to clear the status message after a delay.
func (m Model) clearStatusAfterDelay() tea.Cmd {
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return ClearStatusMsg{}
	})
}

// renderOrphanModal renders the orphaned session modal prompt.
func (m Model) renderOrphanModal() string {
	title := m.orphanedTicket.Title
	if len(title) > 30 {
		title = title[:27] + "..."
	}
	prompt := fmt.Sprintf("Orphaned session found for \"%s\"", title)
	options := "[r]esume  [f]resh  [c]ancel"
	return statusBarStyle.Render(prompt) + "\n" + helpBarStyle.Render(options)
}
