package ticket

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/kareemaly/cortex/internal/cli/sdk"
)

const (
	wideLayoutMinWidth = 100

	// SSE reconnection constants.
	sseInitialBackoff = 2 * time.Second
	sseMaxBackoff     = 30 * time.Second
	pollInterval      = 60 * time.Second
)

// Model is the main Bubbletea model for the ticket detail view.
type Model struct {
	client          *sdk.Client
	ticketID        string
	ticket          *sdk.TicketResponse
	bodyViewport    viewport.Model
	width           int
	height          int
	ready           bool
	loading         bool
	err             error
	showKillModal   bool
	killing         bool
	approving       bool
	spawning        bool
	showOrphanModal bool
	showDeleteModal bool
	embedded        bool // if true, send CloseDetailMsg instead of tea.Quit
	pendingG        bool // tracking 'g' key for 'gg' sequence
	mdRenderer      *glamour.TermRenderer
	executingEdit   bool
	sessions        []sdk.SessionListItem // cached sessions for this project

	// SSE subscription state
	eventCh      <-chan sdk.Event
	cancelEvents context.CancelFunc
	sseBackoff   time.Duration
	sseConnected bool
}

// Message types for async operations.

// TicketLoadedMsg is sent when a ticket is successfully fetched.
type TicketLoadedMsg struct {
	Ticket   *sdk.TicketResponse
	Sessions []sdk.SessionListItem
}

// TicketErrorMsg is sent when fetching a ticket fails.
type TicketErrorMsg struct {
	Err error
}

// SessionKilledMsg is sent when a session is successfully killed.
type SessionKilledMsg struct{}

// SessionKillErrorMsg is sent when killing a session fails.
type SessionKillErrorMsg struct {
	Err error
}

// SessionApprovedMsg is sent when a session is successfully approved.
type SessionApprovedMsg struct{}

// ApproveErrorMsg is sent when approving a session fails.
type ApproveErrorMsg struct {
	Err error
}

// SessionSpawnedMsg is sent when a session is successfully spawned.
type SessionSpawnedMsg struct{}

// SpawnErrorMsg is sent when spawning a session fails.
type SpawnErrorMsg struct {
	Err error
}

// OrphanedSessionMsg is sent when spawn encounters an orphaned session.
type OrphanedSessionMsg struct{}

// SessionDeletedMsg is sent when an orphaned session is successfully deleted.
type SessionDeletedMsg struct{}

// SessionDeleteErrorMsg is sent when deleting a session fails.
type SessionDeleteErrorMsg struct {
	Err error
}

// CloseDetailMsg is sent when user wants to close the detail view.
type CloseDetailMsg struct{}

// RefreshMsg triggers a ticket data reload (used by SSE).
type RefreshMsg struct{}

// EditExecutedMsg is sent when the editor is successfully opened.
type EditExecutedMsg struct{}

// EditErrorMsg is sent when opening the editor fails.
type EditErrorMsg struct {
	Err error
}

// sseConnectedMsg is sent when the SSE connection is established.
type sseConnectedMsg struct {
	ch     <-chan sdk.Event
	cancel context.CancelFunc
}

// EventMsg is sent when an SSE event is received for this ticket.
type EventMsg struct{}

// sseDisconnectedMsg is sent when the SSE connection is lost.
type sseDisconnectedMsg struct{}

// sseReconnectTickMsg is sent when it's time to attempt SSE reconnection.
type sseReconnectTickMsg struct{}

// pollTickMsg is sent periodically as a safety-net data refresh.
type pollTickMsg struct{}

// New creates a new ticket detail model.
func New(client *sdk.Client, ticketID string) Model {
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)
	return Model{
		client:     client,
		ticketID:   ticketID,
		loading:    true,
		mdRenderer: renderer,
	}
}

// NewEmbedded creates a model that sends CloseDetailMsg on close.
func NewEmbedded(client *sdk.Client, ticketID string) Model {
	m := New(client, ticketID)
	m.embedded = true
	return m
}

// TicketID returns the ticket ID this model is displaying.
func (m Model) TicketID() string {
	return m.ticketID
}

// Init initializes the model and starts loading the ticket.
func (m Model) Init() tea.Cmd {
	if m.embedded {
		return tea.Batch(m.loadTicket(), m.startPollTicker())
	}
	return tea.Batch(m.loadTicket(), m.subscribeEvents(), m.startPollTicker())
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		bodyH := m.bodyHeight()

		// Viewport width depends on wide mode.
		vpWidth := m.width
		if m.width >= wideLayoutMinWidth {
			vpWidth = m.width * 70 / 100
		}

		// Update renderer width to match the body panel.
		renderer, _ := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(vpWidth),
		)
		m.mdRenderer = renderer

		if !m.ready {
			m.bodyViewport = viewport.New(vpWidth, bodyH)
			m.bodyViewport.YPosition = 2 // Below header.
			m.ready = true
			if m.ticket != nil {
				m.bodyViewport.SetContent(m.renderBodyContent())
			}
		} else {
			m.bodyViewport.Width = vpWidth
			m.bodyViewport.Height = bodyH
			if m.ticket != nil {
				m.bodyViewport.SetContent(m.renderBodyContent())
			}
		}

		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case TicketLoadedMsg:
		m.loading = false
		m.err = nil
		m.ticket = msg.Ticket
		m.sessions = msg.Sessions
		if m.ready {
			// Preserve scroll position across SSE refreshes.
			savedOffset := m.bodyViewport.YOffset
			m.bodyViewport.SetContent(m.renderBodyContent())
			m.bodyViewport.SetYOffset(savedOffset)
		}
		return m, nil

	case TicketErrorMsg:
		m.loading = false
		m.err = msg.Err
		return m, nil

	case SessionKilledMsg:
		m.killing = false
		m.showKillModal = false
		// Refresh ticket to show updated session state.
		m.loading = true
		return m, m.loadTicket()

	case SessionKillErrorMsg:
		m.killing = false
		m.showKillModal = false
		m.err = msg.Err
		return m, nil

	case SessionApprovedMsg:
		m.approving = false
		// Refresh ticket to show updated state.
		m.loading = true
		return m, m.loadTicket()

	case ApproveErrorMsg:
		m.approving = false
		m.err = msg.Err
		return m, nil

	case SessionSpawnedMsg:
		m.spawning = false
		m.loading = true
		return m, m.loadTicket()

	case SpawnErrorMsg:
		m.spawning = false
		m.err = msg.Err
		return m, nil

	case OrphanedSessionMsg:
		m.spawning = false
		m.showOrphanModal = true
		return m, nil

	case SessionDeletedMsg:
		m.showDeleteModal = false
		m.loading = true
		return m, m.loadTicket()

	case SessionDeleteErrorMsg:
		m.showDeleteModal = false
		m.err = msg.Err
		return m, nil

	case RefreshMsg:
		m.loading = true
		return m, m.loadTicket()

	case sseConnectedMsg:
		// Cancel old connection if replacing.
		if m.cancelEvents != nil {
			m.cancelEvents()
		}
		m.eventCh = msg.ch
		m.cancelEvents = msg.cancel
		m.sseConnected = true
		m.sseBackoff = 0
		return m, tea.Batch(m.loadTicket(), m.waitForEvent())

	case EventMsg:
		return m, tea.Batch(m.loadTicket(), m.waitForEvent())

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
		return m, m.scheduleSSEReconnect()

	case sseReconnectTickMsg:
		return m, m.subscribeEvents()

	case pollTickMsg:
		return m, tea.Batch(m.loadTicket(), m.startPollTicker())

	case EditExecutedMsg:
		m.executingEdit = false
		m.loading = true
		return m, m.loadTicket()

	case EditErrorMsg:
		m.executingEdit = false
		m.err = msg.Err
		return m, nil
	}

	// Handle viewport scroll messages.
	var cmd tea.Cmd
	m.bodyViewport, cmd = m.bodyViewport.Update(msg)
	return m, cmd
}

// handleKeyMsg handles keyboard input.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Modals take priority when visible.
	if m.showDeleteModal {
		return m.handleDeleteModalKey(msg)
	}
	if m.showKillModal {
		return m.handleKillModalKey(msg)
	}
	if m.showOrphanModal {
		return m.handleOrphanModalKey(msg)
	}

	// Quit or close.
	if isKey(msg, KeyQuit, KeyCtrlC) {
		if m.embedded {
			return m, func() tea.Msg { return CloseDetailMsg{} }
		}
		if m.cancelEvents != nil {
			m.cancelEvents()
		}
		return m, tea.Quit
	}

	// Handle Escape for embedded mode.
	if m.embedded && isKey(msg, KeyEscape) {
		return m, func() tea.Msg { return CloseDetailMsg{} }
	}

	// If loading, killing, approving, spawning, or executing edit, don't process other keys.
	if m.loading || m.killing || m.approving || m.spawning || m.executingEdit {
		return m, nil
	}

	// If error, only allow refresh.
	if m.err != nil {
		if isKey(msg, KeyRefresh) {
			m.loading = true
			m.err = nil
			return m, m.loadTicket()
		}
		return m, nil
	}

	// Refresh.
	if isKey(msg, KeyRefresh) {
		m.loading = true
		return m, m.loadTicket()
	}

	// Kill session.
	if isKey(msg, KeyKillSession) {
		if m.hasActiveSession() {
			m.showKillModal = true
		}
		return m, nil
	}

	// Approve session.
	if !m.pendingG && isKey(msg, KeyApprove) {
		if m.hasActiveSession() {
			m.approving = true
			return m, m.approveSession()
		}
		return m, nil
	}

	// Spawn session.
	if isKey(msg, KeySpawn) {
		if m.canSpawn() {
			m.spawning = true
			return m, m.spawnSession()
		}
		return m, nil
	}

	// Edit ticket in $EDITOR.
	if isKey(msg, KeyEdit) {
		m.executingEdit = true
		return m, m.editTicket()
	}

	// Handle 'ga' - focus architect window.
	if m.pendingG && isKey(msg, KeyApprove) {
		m.pendingG = false
		return m, m.focusArchitect()
	}

	// Handle 'G' - jump to bottom.
	if isKey(msg, KeyShiftG) {
		m.pendingG = false
		m.bodyViewport.GotoBottom()
		return m, nil
	}

	// Handle 'g' key for 'gg' sequence.
	if isKey(msg, KeyG) {
		if m.pendingG {
			// Second 'g' - jump to top.
			m.pendingG = false
			m.bodyViewport.GotoTop()
		} else {
			// First 'g' - set pending state.
			m.pendingG = true
		}
		return m, nil
	}

	// Clear pending g on any other key.
	m.pendingG = false

	// Half-page scroll (ctrl+u/d).
	if isKey(msg, KeyCtrlU) {
		m.bodyViewport.HalfPageUp()
		return m, nil
	}
	if isKey(msg, KeyCtrlD) {
		m.bodyViewport.HalfPageDown()
		return m, nil
	}

	// Scroll navigation.
	if isKey(msg, KeyUp, KeyK) {
		m.bodyViewport.ScrollUp(1)
		return m, nil
	}
	if isKey(msg, KeyDown, KeyJ) {
		m.bodyViewport.ScrollDown(1)
		return m, nil
	}
	if isKey(msg, KeyPgUp) {
		m.bodyViewport.PageUp()
		return m, nil
	}
	if isKey(msg, KeyPgDown) {
		m.bodyViewport.PageDown()
		return m, nil
	}
	if isKey(msg, KeyHome) {
		m.bodyViewport.GotoTop()
		return m, nil
	}
	if isKey(msg, KeyEnd) {
		m.bodyViewport.GotoBottom()
		return m, nil
	}

	// Pass to viewport for mouse scroll, etc.
	var cmd tea.Cmd
	m.bodyViewport, cmd = m.bodyViewport.Update(msg)
	return m, cmd
}

// handleKillModalKey handles keyboard input when the kill confirmation modal is shown.
func (m Model) handleKillModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if isKey(msg, KeyYes) {
		m.killing = true
		return m, m.killSession()
	}
	if isKey(msg, KeyNo, KeyEscape) {
		m.showKillModal = false
		return m, nil
	}
	return m, nil
}

// hasActiveSession returns true if there's an active session.
// Sessions are no longer on TicketResponse; we use ticket status as a heuristic.
// The ticket is in "progress" status when an agent is working.
func (m Model) hasActiveSession() bool {
	if m.ticket == nil {
		return false
	}
	return m.ticket.Status == "progress"
}

// killSession returns a command to kill the current session.
func (m Model) killSession() tea.Cmd {
	return func() tea.Msg {
		if m.ticket == nil {
			return SessionKillErrorMsg{Err: fmt.Errorf("no session to kill")}
		}
		// Use ticket ID prefix (short ID) to find session
		err := m.client.KillSession(m.ticket.ID[:8])
		if err != nil {
			return SessionKillErrorMsg{Err: err}
		}
		return SessionKilledMsg{}
	}
}

// approveSession returns a command to approve the current session.
func (m Model) approveSession() tea.Cmd {
	return func() tea.Msg {
		if m.ticket == nil {
			return ApproveErrorMsg{Err: fmt.Errorf("no session to approve")}
		}
		// Use ticket ID prefix (short ID) to find session
		err := m.client.ApproveSession(m.ticket.ID[:8])
		if err != nil {
			return ApproveErrorMsg{Err: err}
		}
		return SessionApprovedMsg{}
	}
}

// canSpawn returns true when the ticket can have a session spawned.
func (m Model) canSpawn() bool {
	if m.ticket == nil {
		return false
	}
	// Can spawn if in backlog or progress with no active session.
	status := m.ticket.Status
	if status != "backlog" && status != "progress" {
		return false
	}
	return !m.hasActiveSession()
}

// editTicket returns a command to open the ticket in $EDITOR via tmux popup.
func (m Model) editTicket() tea.Cmd {
	return func() tea.Msg {
		if m.ticket == nil {
			return EditErrorMsg{Err: fmt.Errorf("no ticket")}
		}
		err := m.client.EditTicket(m.ticket.ID)
		if err != nil {
			return EditErrorMsg{Err: err}
		}
		return EditExecutedMsg{}
	}
}

// spawnSession returns a command to spawn a session for the current ticket.
func (m Model) spawnSession() tea.Cmd {
	return func() tea.Msg {
		if m.ticket == nil {
			return SpawnErrorMsg{Err: fmt.Errorf("no ticket to spawn")}
		}
		_, err := m.client.SpawnSession(m.ticket.Status, m.ticket.ID, "normal")
		if err != nil {
			if apiErr, ok := err.(*sdk.APIError); ok && apiErr.IsOrphanedSession() {
				return OrphanedSessionMsg{}
			}
			return SpawnErrorMsg{Err: err}
		}
		return SessionSpawnedMsg{}
	}
}

// spawnSessionWithMode returns a command to spawn a session with a specific mode.
func (m Model) spawnSessionWithMode(mode string) tea.Cmd {
	return func() tea.Msg {
		if m.ticket == nil {
			return SpawnErrorMsg{Err: fmt.Errorf("no ticket to spawn")}
		}
		_, err := m.client.SpawnSession(m.ticket.Status, m.ticket.ID, mode)
		if err != nil {
			return SpawnErrorMsg{Err: err}
		}
		return SessionSpawnedMsg{}
	}
}

// focusArchitect returns a command to focus the architect tmux window.
func (m Model) focusArchitect() tea.Cmd {
	return func() tea.Msg {
		_ = m.client.FocusArchitect()
		return nil
	}
}

// handleOrphanModalKey handles keyboard input when the orphan modal is shown.
func (m Model) handleOrphanModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if isKey(msg, KeyRefresh) { // 'r' for resume
		m.showOrphanModal = false
		m.spawning = true
		return m, m.spawnSessionWithMode("resume")
	}
	if isKey(msg, KeyFresh) { // 'f' for fresh
		m.showOrphanModal = false
		m.spawning = true
		return m, m.spawnSessionWithMode("fresh")
	}
	if isKey(msg, KeyDeleteOrphan) { // 'D' for delete
		m.showOrphanModal = false
		m.showDeleteModal = true
		return m, nil
	}
	if isKey(msg, KeyCancel, KeyEscape) {
		m.showOrphanModal = false
		return m, nil
	}
	return m, nil
}

// handleDeleteModalKey handles keyboard input when the delete confirmation modal is shown.
func (m Model) handleDeleteModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if isKey(msg, KeyYes) { // 'y' for yes - delete the session
		return m, m.deleteOrphanedSession()
	}
	if isKey(msg, KeyNo, KeyEscape) { // 'n' or Esc for no - go back to orphan modal
		m.showDeleteModal = false
		m.showOrphanModal = true
		return m, nil
	}
	return m, nil
}

// deleteOrphanedSession returns a command to delete an orphaned session.
func (m Model) deleteOrphanedSession() tea.Cmd {
	return func() tea.Msg {
		if m.ticket == nil {
			return SessionDeleteErrorMsg{Err: fmt.Errorf("no session to delete")}
		}
		// Use ticket ID prefix (short ID) to find session
		if err := m.client.KillSession(m.ticket.ID[:8]); err != nil {
			return SessionDeleteErrorMsg{Err: err}
		}
		return SessionDeletedMsg{}
	}
}

// renderOrphanModal renders the orphaned session modal.
func (m Model) renderOrphanModal() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(warningStyle.Render("Orphaned session detected"))
	b.WriteString("\n\n")
	b.WriteString("The tmux window for this session was closed.\n\n")
	b.WriteString("[r]esume  [f]resh  [D]elete  [c]ancel")

	return b.String()
}

// renderDeleteModal renders the delete confirmation modal.
func (m Model) renderDeleteModal() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(warningStyle.Render("Delete orphaned session?"))
	b.WriteString("\n\n")
	b.WriteString("This will end the session record. You can spawn a new one later.\n\n")
	b.WriteString("[y]es  [n]o")

	return b.String()
}

// View renders the ticket detail view.
func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	var b strings.Builder

	// Header.
	header := m.renderHeader()
	b.WriteString(header)
	b.WriteString("\n")

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
		b.WriteString(loadingStyle.Render("Loading ticket..."))
		return b.String()
	}

	// Handle killing state.
	if m.killing {
		b.WriteString(loadingStyle.Render("Killing session..."))
		return b.String()
	}

	// Handle approving state.
	if m.approving {
		b.WriteString(loadingStyle.Render("Approving session..."))
		return b.String()
	}

	// Handle spawning state.
	if m.spawning {
		b.WriteString(loadingStyle.Render("Spawning session..."))
		return b.String()
	}

	// Handle executing edit state.
	if m.executingEdit {
		b.WriteString(loadingStyle.Render("Opening editor..."))
		return b.String()
	}

	// Kill confirmation modal.
	if m.showKillModal {
		b.WriteString(m.renderKillModal())
		return b.String()
	}

	// Orphan session modal.
	if m.showOrphanModal {
		b.WriteString(m.renderOrphanModal())
		return b.String()
	}

	// Delete confirmation modal.
	if m.showDeleteModal {
		b.WriteString(m.renderDeleteModal())
		return b.String()
	}

	// Body + attributes.
	bodyH := m.bodyHeight()
	b.WriteString(m.renderRow1(bodyH))

	// Help bar.
	b.WriteString("\n")
	b.WriteString(helpBarStyle.Render(helpText(
		int(m.bodyViewport.ScrollPercent()*100),
		m.hasActiveSession(), m.canSpawn(),
		m.embedded,
	)))

	return b.String()
}

// renderKillModal renders the kill session confirmation modal.
func (m Model) renderKillModal() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(warningStyle.Render("Kill active session?"))
	b.WriteString("\n\n")
	b.WriteString("This will terminate the agent session and close the tmux window.\n\n")
	b.WriteString("[y]es  [n]o")

	return b.String()
}

// renderHeader renders the fixed header bar.
func (m Model) renderHeader() string {
	if m.ticket == nil {
		return headerStyle.Render("Loading...")
	}

	// ID + [Type] + Title + Status badge.
	id := ticketIDStyle.Render(m.ticket.ID[:8])

	// Build type badge for non-work types (consistent with kanban).
	typeBadge := ""
	if m.ticket.Type != "" && m.ticket.Type != "work" {
		typeBadge = typeBadgeStyle(m.ticket.Type).Render("[" + m.ticket.Type + "]")
	}

	title := titleStyle.Render(m.ticket.Title)
	status := statusStyle(m.ticket.Status).Render(m.ticket.Status)

	// Build left side: id + type badge (if any) + title.
	left := id
	if typeBadge != "" {
		left += " " + typeBadge
	}
	left += " " + title
	right := status

	padding := max(m.width-lipgloss.Width(left)-lipgloss.Width(right)-2, 1)
	return left + strings.Repeat(" ", padding) + right
}

// bodyHeight computes the available height for the body viewport.
func (m Model) bodyHeight() int {
	// available = height - 3 (header with padding + help bar)
	available := max(m.height-3, 2)
	return available
}

// renderRow1 renders the body viewport + attributes panel (wide) or body only (narrow).
func (m Model) renderRow1(height int) string {
	if m.width >= wideLayoutMinWidth {
		// Wide: body viewport | divider | attributes panel.
		bodyWidth := m.width * 70 / 100
		attrWidth := m.width - bodyWidth - 1

		body := m.bodyViewport.View()
		bodyBlock := lipgloss.NewStyle().
			Width(bodyWidth).
			Height(height).
			Render(body)

		divider := m.renderPanelDivider(height)
		attrs := m.renderAttributes(attrWidth, height)

		return lipgloss.JoinHorizontal(lipgloss.Top, bodyBlock, divider, attrs)
	}

	// Narrow: full-width body viewport only.
	return m.bodyViewport.View()
}

// renderAttributes renders the DETAILS + SESSION sections for the attributes panel.
func (m Model) renderAttributes(width, height int) string {
	if m.ticket == nil {
		return ""
	}

	var b strings.Builder

	// DETAILS section.
	b.WriteString(attributeHeaderStyle.Render("DETAILS"))
	b.WriteString("\n")

	b.WriteString(attributeLabelStyle.Render("Created  "))
	b.WriteString(attributeValueStyle.Render(m.ticket.Created.Format("Jan 02, 15:04")))

	b.WriteString("\n")
	b.WriteString(attributeLabelStyle.Render("Updated  "))
	b.WriteString(attributeValueStyle.Render(m.ticket.Updated.Format("Jan 02, 15:04")))

	if m.ticket.Due != nil {
		b.WriteString("\n")
		b.WriteString(attributeLabelStyle.Render("Due      "))
		// Color-code based on urgency
		now := time.Now()
		dueDateStyle := attributeValueStyle
		if m.ticket.Due.Before(now) {
			dueDateStyle = overdueStyle
		} else if m.ticket.Due.Before(now.Add(24 * time.Hour)) {
			dueDateStyle = dueSoonStyle
		}
		b.WriteString(dueDateStyle.Render(m.ticket.Due.Format("Jan 02, 15:04")))
	}

	// SESSION section
	b.WriteString("\n\n")
	b.WriteString(attributeHeaderStyle.Render("SESSION"))
	b.WriteString("\n")
	if m.sessions != nil {
		found := false
		for _, s := range m.sessions {
			if s.TicketID == m.ticket.ID {
				found = true
				b.WriteString(attributeLabelStyle.Render("Agent    "))
				b.WriteString(attributeValueStyle.Render(s.Agent))
				b.WriteString("\n")
				b.WriteString(attributeLabelStyle.Render("Status   "))
				b.WriteString(attributeValueStyle.Render(s.Status))
				if s.Tool != nil && *s.Tool != "" {
					b.WriteString("\n")
					b.WriteString(attributeLabelStyle.Render("Tool     "))
					b.WriteString(attributeValueStyle.Render(*s.Tool))
				}
				break
			}
		}
		if !found {
			b.WriteString(attributeLabelStyle.Render("No active session"))
		}
	} else {
		b.WriteString(attributeLabelStyle.Render("No active session"))
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		PaddingLeft(1).
		Render(b.String())
}

// renderBodyContent returns the content for the body viewport.
// In wide mode: just the markdown body. In narrow mode: dates + body + session.
func (m Model) renderBodyContent() string {
	if m.ticket == nil {
		return ""
	}

	if m.width >= wideLayoutMinWidth {
		// Wide mode: just body markdown.
		if m.ticket.Body == "" {
			return ""
		}
		return m.renderMarkdown(m.ticket.Body)
	}

	// Narrow mode: inline dates + body.
	var b strings.Builder

	// Inline dates.
	b.WriteString(labelStyle.Render("Created: "))
	b.WriteString(valueStyle.Render(m.ticket.Created.Format("Jan 02, 2006 15:04")))
	b.WriteString("  ")
	b.WriteString(labelStyle.Render("Updated: "))
	b.WriteString(valueStyle.Render(m.ticket.Updated.Format("Jan 02, 2006 15:04")))

	// Body.
	if m.ticket.Body != "" {
		b.WriteString("\n\n")
		b.WriteString(m.renderMarkdown(m.ticket.Body))
	}

	return b.String()
}

// renderPanelDivider renders a vertical divider column.
func (m Model) renderPanelDivider(height int) string {
	var b strings.Builder
	for i := 0; i < height; i++ {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(dividerStyle.Render("│"))
	}
	return b.String()
}

// renderMarkdown renders content as markdown using glamour.
func (m Model) renderMarkdown(content string) string {
	if m.mdRenderer == nil {
		return content
	}
	rendered, err := m.mdRenderer.Render(content)
	if err != nil {
		return content // fallback to raw on error
	}
	return strings.TrimSpace(rendered)
}

// loadTicket returns a command to load the ticket and sessions.
func (m Model) loadTicket() tea.Cmd {
	return func() tea.Msg {
		ticket, err := m.client.FindTicketByID(m.ticketID)
		if err != nil {
			return TicketErrorMsg{Err: err}
		}
		// Fetch sessions (non-fatal if it fails)
		var sessions []sdk.SessionListItem
		if resp, err := m.client.ListSessions(); err == nil {
			sessions = resp.Sessions
		}
		return TicketLoadedMsg{Ticket: ticket, Sessions: sessions}
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

// waitForEvent returns a command that waits for the next SSE event relevant to this ticket.
func (m Model) waitForEvent() tea.Cmd {
	if m.eventCh == nil {
		return nil
	}
	ch := m.eventCh
	ticketID := m.ticketID
	return func() tea.Msg {
		for event := range ch {
			if event.TicketID == ticketID {
				return EventMsg{}
			}
		}
		return sseDisconnectedMsg{}
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
