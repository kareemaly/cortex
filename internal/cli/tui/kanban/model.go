package kanban

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/cli/tui/tuilog"
	"github.com/kareemaly/cortex/internal/cli/tui/variant"
)

// SSE reconnection constants.
const (
	sseInitialBackoff = 2 * time.Second
	sseMaxBackoff     = 30 * time.Second
	pollInterval      = 60 * time.Second
)

// Model is the main Bubbletea model for the kanban board.
type Model struct {
	columns       [3]Column
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

	// Delete confirmation modal state
	showDeleteModal bool

	// Variant selector state
	showVariantSelector bool
	variantSelector     variant.Model
	pendingSpawnTicket  *sdk.TicketSummary
	pendingSpawnMode    string
	pendingSpawnVariant string

	// Vim navigation state
	pendingG bool // tracking 'g' key for 'gg' sequence

	// SSE subscription state
	eventCh      <-chan sdk.Event
	cancelEvents context.CancelFunc
	sseBackoff   time.Duration
	sseConnected bool

	// Log viewer state
	logBuf        *tuilog.Buffer
	logViewer     tuilog.Viewer
	showLogViewer bool
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

type SessionSpawnedMsg struct {
	Session  *sdk.SessionResponse
	Ticket   *sdk.TicketSummary
	Queued   bool
	Position int
}

type SessionErrorMsg struct {
	Err error
}

type DequeueMsg struct {
	Ticket *sdk.TicketSummary
}

type DequeueErrorMsg struct {
	Err error
}

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

// SessionDeletedMsg is sent when an orphaned session is successfully deleted.
type SessionDeletedMsg struct {
	Ticket *sdk.TicketSummary
}

// SessionDeleteErrorMsg is sent when deleting a session fails.
type SessionDeleteErrorMsg struct {
	Err error
}

// openEditorMsg is sent when the editor popup was launched successfully.
type openEditorMsg struct{}

// openEditorErrMsg is sent when launching the editor popup fails.
type openEditorErrMsg struct{ Err error }

// sseConnectedMsg is sent when the SSE connection is established.
type sseConnectedMsg struct {
	ch     <-chan sdk.Event
	cancel context.CancelFunc
}

// EventMsg is sent when an SSE event is received.
type EventMsg struct{}

// sseDisconnectedMsg is sent when the SSE connection is lost.
type sseDisconnectedMsg struct{}

// sseReconnectTickMsg is sent when it's time to attempt SSE reconnection.
type sseReconnectTickMsg struct{}

// pollTickMsg is sent periodically as a safety-net data refresh.
type pollTickMsg struct{}

// variantsLoadedMsg is sent when agent variants are fetched successfully.
type variantsLoadedMsg struct{ variants []string }

// variantsErrMsg is sent when fetching agent variants fails.
type variantsErrMsg struct{ err error }

// New creates a new kanban model with the given client and log buffer.
func New(client *sdk.Client, logBuf *tuilog.Buffer) Model {
	return Model{
		columns: [3]Column{
			NewColumn("Backlog", "backlog"),
			NewColumn("Progress", "progress"),
			NewColumn("Done", "done"),
		},
		client:    client,
		loading:   true,
		logBuf:    logBuf,
		logViewer: tuilog.NewViewer(logBuf),
	}
}

// Init initializes the model and starts loading tickets.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadTickets(), m.subscribeEvents(), m.startPollTicker())
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle log viewer dismiss.
	if _, ok := msg.(tuilog.DismissLogViewerMsg); ok {
		m.showLogViewer = false
		return m, nil
	}

	// Delegate to log viewer when active.
	if m.showLogViewer {
		if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
			m.width = sizeMsg.Width
			m.height = sizeMsg.Height
			m.ready = true
			m.logViewer.SetSize(m.width, m.height)
		}
		var cmd tea.Cmd
		m.logViewer, cmd = m.logViewer.Update(msg)
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
		m.columns[2].SetTickets(msg.Response.Done)
		m.logBuf.Debug("api", "tickets loaded")
		return m, nil

	case TicketsErrorMsg:
		m.loading = false
		m.err = msg.Err
		m.logBuf.Errorf("api", "failed to load tickets: %s", msg.Err)
		return m, nil

	case SessionSpawnedMsg:
		if msg.Queued {
			m.statusMsg = fmt.Sprintf("Queued #%d: %s", msg.Position, msg.Ticket.Title)
			m.statusIsError = false
			m.logBuf.Infof("queue", "ticket queued #%d: %s", msg.Position, msg.Ticket.Title)
		} else {
			m.statusMsg = fmt.Sprintf("Session spawned for: %s", msg.Ticket.Title)
			m.statusIsError = false
			m.logBuf.Infof("spawn", "session spawned for: %s", msg.Ticket.Title)
		}
		return m, tea.Batch(m.loadTickets(), m.clearStatusAfterDelay())

	case SessionErrorMsg:
		m.statusMsg = fmt.Sprintf("Error: %s", msg.Err)
		m.statusIsError = true
		m.logBuf.Errorf("spawn", "spawn failed: %s", msg.Err)
		return m, m.clearStatusAfterDelay()

	case DequeueMsg:
		m.statusMsg = fmt.Sprintf("Dequeued: %s", msg.Ticket.Title)
		m.statusIsError = false
		m.logBuf.Infof("queue", "dequeued: %s", msg.Ticket.Title)
		return m, tea.Batch(m.loadTickets(), m.clearStatusAfterDelay())

	case DequeueErrorMsg:
		m.statusMsg = fmt.Sprintf("Dequeue error: %s", msg.Err)
		m.statusIsError = true
		m.logBuf.Errorf("queue", "dequeue failed: %s", msg.Err)
		return m, m.clearStatusAfterDelay()

	case OrphanedSessionMsg:
		m.showOrphanModal = true
		m.orphanedTicket = msg.Ticket
		m.statusMsg = ""
		m.logBuf.Warnf("spawn", "orphaned session for: %s", msg.Ticket.Title)
		return m, nil

	case FocusSuccessMsg:
		m.statusMsg = fmt.Sprintf("Focused: %s", msg.Window)
		m.statusIsError = false
		m.logBuf.Infof("focus", "focused: %s", msg.Window)
		return m, m.clearStatusAfterDelay()

	case FocusErrorMsg:
		m.statusMsg = fmt.Sprintf("Focus error: %s", msg.Err)
		m.statusIsError = true
		m.logBuf.Errorf("focus", "focus failed: %s", msg.Err)
		return m, m.clearStatusAfterDelay()

	case openEditorMsg:
		m.statusMsg = "Editor opened"
		m.statusIsError = false
		return m, m.clearStatusAfterDelay()

	case openEditorErrMsg:
		m.statusMsg = msg.Err.Error()
		m.statusIsError = true
		return m, m.clearStatusAfterDelay()

	case SessionDeletedMsg:
		m.showDeleteModal = false
		m.orphanedTicket = nil
		m.statusMsg = fmt.Sprintf("Session deleted for: %s", msg.Ticket.Title)
		m.statusIsError = false
		m.logBuf.Infof("delete", "session deleted for: %s", msg.Ticket.Title)
		return m, tea.Batch(m.loadTickets(), m.clearStatusAfterDelay())

	case SessionDeleteErrorMsg:
		m.showDeleteModal = false
		m.statusMsg = fmt.Sprintf("Delete error: %s", msg.Err)
		m.statusIsError = true
		m.logBuf.Errorf("delete", "delete failed: %s", msg.Err)
		return m, m.clearStatusAfterDelay()

	case ClearStatusMsg:
		m.statusMsg = ""
		m.statusIsError = false
		return m, nil

	case sseConnectedMsg:
		// Cancel old connection if replacing.
		if m.cancelEvents != nil {
			m.cancelEvents()
		}
		m.eventCh = msg.ch
		m.cancelEvents = msg.cancel
		m.sseConnected = true
		m.sseBackoff = 0
		m.logBuf.Info("sse", "connected to event stream")
		return m, tea.Batch(m.loadTickets(), m.waitForEvent())

	case EventMsg:
		m.logBuf.Debug("sse", "event received")
		return m, tea.Batch(m.loadTickets(), m.waitForEvent())

	case sseDisconnectedMsg:
		// Guard: if sseConnected is true, this is from a replaced connection; ignore.
		if m.sseConnected {
			m.sseConnected = false
			return m, nil
		}
		m.eventCh = nil
		if m.cancelEvents != nil {
			m.cancelEvents()
			m.cancelEvents = nil
		}
		m.sseBackoff = nextBackoff(m.sseBackoff)
		m.logBuf.Warnf("sse", "disconnected, reconnecting in %s", m.sseBackoff)
		return m, m.scheduleSSEReconnect()

	case sseReconnectTickMsg:
		m.logBuf.Debug("sse", "attempting reconnect")
		return m, m.subscribeEvents()

	case pollTickMsg:
		return m, tea.Batch(m.loadTickets(), m.startPollTicker())

	case variantsLoadedMsg:
		if len(msg.variants) == 1 {
			m.pendingSpawnVariant = msg.variants[0]
			return m, m.spawnSessionWithVariant(m.pendingSpawnTicket, m.pendingSpawnMode, msg.variants[0])
		}
		m.variantSelector = variant.New("Select agent variant", msg.variants)
		m.showVariantSelector = true
		return m, nil

	case variantsErrMsg:
		m.statusMsg = fmt.Sprintf("Error loading variants: %s", msg.err)
		m.statusIsError = true
		m.pendingSpawnTicket = nil
		m.pendingSpawnMode = ""
		return m, m.clearStatusAfterDelay()

	case variant.SelectedMsg:
		m.showVariantSelector = false
		ticket := m.pendingSpawnTicket
		mode := m.pendingSpawnMode
		m.pendingSpawnVariant = msg.Name // preserve for potential orphan re-spawn
		m.pendingSpawnTicket = nil
		m.pendingSpawnMode = ""
		return m, m.spawnSessionWithVariant(ticket, mode, msg.Name)

	case variant.CancelledMsg:
		m.showVariantSelector = false
		m.pendingSpawnTicket = nil
		m.pendingSpawnMode = ""
		m.statusMsg = "Spawn cancelled"
		m.statusIsError = false
		return m, m.clearStatusAfterDelay()
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

	// Toggle log viewer.
	if isKey(msg, KeyExclaim) {
		m.showLogViewer = !m.showLogViewer
		if m.showLogViewer {
			m.logViewer.SetSize(m.width, m.height)
			m.logViewer.Reset()
		}
		return m, nil
	}

	// Modal state takes priority.
	if m.showVariantSelector {
		return m.handleVariantSelectorKey(msg)
	}
	if m.showDeleteModal {
		return m.handleDeleteModalKey(msg)
	}
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
		if m.activeColumn < 2 {
			m.activeColumn++
		}
		return m, nil
	}

	// Spawn session.
	if isKey(msg, KeySpawn) {
		t := m.columns[m.activeColumn].SelectedTicket()
		if t != nil {
			m.pendingSpawnTicket = t
			m.pendingSpawnMode = "normal"
			m.statusMsg = fmt.Sprintf("Spawning session for: %s...", t.Title)
			m.statusIsError = false
			return m, m.loadVariants()
		}
		return m, nil
	}

	// Dequeue ticket.
	if isKey(msg, KeyDequeue) {
		t := m.columns[m.activeColumn].SelectedTicket()
		if t != nil && t.QueuePosition != nil {
			m.statusMsg = fmt.Sprintf("Dequeuing: %s...", t.Title)
			m.statusIsError = false
			return m, m.dequeueTicket(t)
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

	// Refresh.
	if isKey(msg, KeyRefresh) {
		m.loading = true
		return m, m.loadTickets()
	}

	// Open ticket in editor.
	if isKey(msg, KeyOpenEditor, KeyEnter) {
		t := m.columns[m.activeColumn].SelectedTicket()
		if t != nil {
			m.statusMsg = "Opening editor..."
			m.statusIsError = false
			return m, m.openTicketInEditor(t)
		}
		return m, nil
	}

	return m, nil
}

// handleOrphanModalKey handles keyboard input when the orphan modal is shown.
func (m Model) handleOrphanModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case isKey(msg, KeyRefresh): // 'r' for resume
		m.showOrphanModal = false
		ticket := m.orphanedTicket
		variantName := m.pendingSpawnVariant
		m.orphanedTicket = nil
		m.pendingSpawnVariant = ""
		m.statusMsg = fmt.Sprintf("Resuming session for: %s...", ticket.Title)
		m.statusIsError = false
		return m, m.spawnSessionWithVariant(ticket, "resume", variantName)

	case isKey(msg, KeyFresh): // 'f' for fresh
		m.showOrphanModal = false
		ticket := m.orphanedTicket
		variantName := m.pendingSpawnVariant
		m.orphanedTicket = nil
		m.pendingSpawnVariant = ""
		m.statusMsg = fmt.Sprintf("Starting fresh session for: %s...", ticket.Title)
		m.statusIsError = false
		return m, m.spawnSessionWithVariant(ticket, "fresh", variantName)

	case isKey(msg, KeyDeleteOrphan): // 'D' for delete
		m.showOrphanModal = false
		m.showDeleteModal = true
		return m, nil

	case isKey(msg, KeyCancel, KeyEscape): // 'c' or Esc for cancel
		m.showOrphanModal = false
		m.orphanedTicket = nil
		m.statusMsg = "Spawn cancelled"
		m.statusIsError = false
		return m, m.clearStatusAfterDelay()
	}
	return m, nil
}

// handleDeleteModalKey handles keyboard input when the delete confirmation modal is shown.
func (m Model) handleDeleteModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case isKey(msg, KeyYes): // 'y' for yes - delete the session
		m.statusMsg = "Deleting session..."
		m.statusIsError = false
		return m, m.deleteOrphanedSession(m.orphanedTicket)

	case isKey(msg, KeyNo, KeyEscape): // 'n' or Esc for no - go back to orphan modal
		m.showDeleteModal = false
		m.showOrphanModal = true
		return m, nil
	}
	return m, nil
}

// deleteOrphanedSession returns a command to delete an orphaned session.
func (m Model) deleteOrphanedSession(ticket *sdk.TicketSummary) tea.Cmd {
	return func() tea.Msg {
		// Use ticket ID prefix (short ID) to find and kill session
		if err := m.client.KillSession(ticket.ID[:8]); err != nil {
			return SessionDeleteErrorMsg{Err: err}
		}
		return SessionDeletedMsg{Ticket: ticket}
	}
}

// View renders the kanban board.
func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	// Log viewer overlay.
	if m.showLogViewer {
		return m.logViewer.View()
	}

	var b strings.Builder

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
	columnWidth := max((m.width-2)/3, 20) // -2 for minimal side margins

	// Calculate available height for columns.
	// Status bar (1) + help bar (1) + margins (2) = ~4 lines overhead
	columnHeight := max(m.height-4, 5)

	// Render columns side by side.
	cols := make([]string, 3)
	for i := range m.columns {
		cols[i] = m.columns[i].View(columnWidth, i == m.activeColumn, columnHeight)
	}
	columnsView := lipgloss.JoinHorizontal(lipgloss.Top, cols...)

	// All modals render as centered overlays.
	if m.showVariantSelector {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.variantSelector.View())
	}
	if m.showOrphanModal {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.renderOrphanModal())
	}
	if m.showDeleteModal {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.renderDeleteModal())
	}

	b.WriteString(columnsView)
	b.WriteString("\n")

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
	help := helpBarStyle.Render(helpText())
	badge := m.logBadge()
	if badge != "" {
		help = help + "  " + badge
	}
	b.WriteString(help)

	return b.String()
}

// loadTickets returns a command to load all tickets.
func (m Model) loadTickets() tea.Cmd {
	return func() tea.Msg {
		resp, err := m.client.ListAllTickets("", nil)
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
			return sseDisconnectedMsg{}
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
			return sseDisconnectedMsg{}
		}
		return EventMsg{}
	}
}

// nextBackoff doubles the current backoff duration, capped at sseMaxBackoff.
func nextBackoff(current time.Duration) time.Duration {
	if current == 0 {
		return sseInitialBackoff
	}
	next := current * 2
	if next > sseMaxBackoff {
		return sseMaxBackoff
	}
	return next
}

// scheduleSSEReconnect returns a command that fires after the current backoff delay.
func (m Model) scheduleSSEReconnect() tea.Cmd {
	return tea.Tick(m.sseBackoff, func(time.Time) tea.Msg {
		return sseReconnectTickMsg{}
	})
}

// startPollTicker returns a command that fires after the poll interval.
func (m Model) startPollTicker() tea.Cmd {
	return tea.Tick(pollInterval, func(time.Time) tea.Msg {
		return pollTickMsg{}
	})
}

// loadVariants fetches available agent variants from the daemon.
func (m Model) loadVariants() tea.Cmd {
	return func() tea.Msg {
		variants, err := m.client.GetVariants()
		if err != nil {
			return variantsErrMsg{err: err}
		}
		return variantsLoadedMsg{variants: variants}
	}
}

// spawnSessionWithVariant spawns a session using a resolved variant name.
func (m Model) spawnSessionWithVariant(ticket *sdk.TicketSummary, mode, variantName string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.SpawnSession(ticket.Status, ticket.ID, mode, variantName)
		if err != nil {
			if apiErr, ok := err.(*sdk.APIError); ok && apiErr.IsOrphanedSession() {
				return OrphanedSessionMsg{Ticket: ticket}
			}
			return SessionErrorMsg{Err: err}
		}
		return SessionSpawnedMsg{Session: result.Session, Ticket: ticket, Queued: result.Queued, Position: result.Position}
	}
}

// handleVariantSelectorKey delegates key events to the variant selector popup.
func (m Model) handleVariantSelectorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.variantSelector, cmd = m.variantSelector.Update(msg)
	return m, cmd
}

func (m Model) dequeueTicket(ticket *sdk.TicketSummary) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.Dequeue(ticket.ID); err != nil {
			return DequeueErrorMsg{Err: err}
		}
		return DequeueMsg{Ticket: ticket}
	}
}

func (m Model) focusTicket(ticket *sdk.TicketSummary) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.FocusTicket(ticket.ID); err != nil {
			return FocusErrorMsg{Err: err}
		}
		return FocusSuccessMsg{Window: ticket.Title}
	}
}

// openTicketInEditor returns a command to open the ticket's index.md in $EDITOR
// via a tmux popup.
func (m Model) openTicketInEditor(ticket *sdk.TicketSummary) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.EditTicket(ticket.ID); err != nil {
			return openEditorErrMsg{Err: fmt.Errorf("open editor: %w", err)}
		}
		return openEditorMsg{}
	}
}

// clearStatusAfterDelay returns a command to clear the status message after a delay.
func (m Model) clearStatusAfterDelay() tea.Cmd {
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return ClearStatusMsg{}
	})
}

// logBadge renders an error/warning count badge for the status bar.
func (m Model) logBadge() string {
	ec := m.logBuf.ErrorCount()
	wc := m.logBuf.WarnCount()
	if ec == 0 && wc == 0 {
		return ""
	}
	var parts []string
	if ec > 0 {
		parts = append(parts, errorStatusStyle.Render(fmt.Sprintf("E:%d", ec)))
	}
	if wc > 0 {
		parts = append(parts, warnBadgeStyle.Render(fmt.Sprintf("W:%d", wc)))
	}
	return strings.Join(parts, " ")
}

var modalBorderStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("214")).
	Padding(1, 2)

var modalTitleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("255")).
	MarginBottom(1)

var modalHelpStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("240")).
	MarginTop(1)

// renderOrphanModal renders the orphaned session modal as a popup box.
func (m Model) renderOrphanModal() string {
	title := m.orphanedTicket.Title
	if len(title) > 40 {
		title = title[:37] + "..."
	}
	content := modalTitleStyle.Render("Orphaned Session") + "\n" +
		lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render("\""+title+"\"") + "\n" +
		modalHelpStyle.Render("[r] resume   [f] fresh   [D] delete   [esc] cancel")
	return modalBorderStyle.Render(content)
}

// renderDeleteModal renders the delete confirmation modal as a popup box.
func (m Model) renderDeleteModal() string {
	title := m.orphanedTicket.Title
	if len(title) > 40 {
		title = title[:37] + "..."
	}
	content := modalTitleStyle.Render("Delete Session?") + "\n" +
		lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render("\""+title+"\"") + "\n" +
		modalHelpStyle.Render("[y] yes   [n] back")
	return modalBorderStyle.Render(content)
}
