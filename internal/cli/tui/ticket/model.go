package ticket

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/kareemaly/cortex/internal/cli/sdk"
)

const wideLayoutMinWidth = 100

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
	embedded        bool // if true, send CloseDetailMsg instead of tea.Quit
	pendingG        bool // tracking 'g' key for 'gg' sequence
	mdRenderer      *glamour.TermRenderer
	focusedRow      int // 0=Row1 (body), 1=Row2 (comments)
	commentCursor   int // cursor index in ticket.Comments
	showDetailModal bool
	modalViewport   viewport.Model
	modalCommentIdx int // index into ticket.Comments
	rejecting       bool
	executingDiff   bool

	// SSE subscription state
	eventCh      <-chan sdk.Event
	cancelEvents context.CancelFunc
}

// Message types for async operations.

// TicketLoadedMsg is sent when a ticket is successfully fetched.
type TicketLoadedMsg struct {
	Ticket *sdk.TicketResponse
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

// CloseDetailMsg is sent when user wants to close the detail view.
type CloseDetailMsg struct{}

// SessionRejectedMsg is sent when a session is successfully rejected.
type SessionRejectedMsg struct{}

// RejectErrorMsg is sent when rejecting a session fails.
type RejectErrorMsg struct {
	Err error
}

// RefreshMsg triggers a ticket data reload (used by SSE).
type RefreshMsg struct{}

// DiffExecutedMsg is sent when a diff action is successfully executed.
type DiffExecutedMsg struct{}

// DiffErrorMsg is sent when executing a diff action fails.
type DiffErrorMsg struct {
	Err error
}

// sseConnectedMsg is sent when the SSE connection is established.
type sseConnectedMsg struct {
	ch     <-chan sdk.Event
	cancel context.CancelFunc
}

// EventMsg is sent when an SSE event is received for this ticket.
type EventMsg struct{}

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
		return m.loadTicket()
	}
	return tea.Batch(m.loadTicket(), m.subscribeEvents())
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		row1H, _ := m.rowHeights()

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
			m.bodyViewport = viewport.New(vpWidth, row1H)
			m.bodyViewport.YPosition = 2 // Below header.
			m.ready = true
			if m.ticket != nil {
				m.bodyViewport.SetContent(m.renderBodyContent())
			}
		} else {
			m.bodyViewport.Width = vpWidth
			m.bodyViewport.Height = row1H
			if m.ticket != nil {
				m.bodyViewport.SetContent(m.renderBodyContent())
			}
		}

		// Resize modal viewport if open.
		if m.showDetailModal {
			modalInnerWidth := max(m.width*60/100-6, 20)
			modalInnerHeight := max(m.height*70/100-7, 5)
			m.modalViewport.Width = modalInnerWidth
			m.modalViewport.Height = modalInnerHeight
			m.modalViewport.SetContent(m.renderModalContent(modalInnerWidth))
		}

		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case TicketLoadedMsg:
		m.loading = false
		m.err = nil
		m.ticket = msg.Ticket
		if m.ready {
			m.bodyViewport.SetContent(m.renderBodyContent())
			m.bodyViewport.GotoTop()
		}
		// Clamp comment cursor to valid range.
		if count := len(m.ticket.Comments); count > 0 {
			m.commentCursor = min(m.commentCursor, count-1)
		} else {
			m.commentCursor = 0
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

	case SessionRejectedMsg:
		m.rejecting = false
		m.loading = true
		return m, m.loadTicket()

	case RejectErrorMsg:
		m.rejecting = false
		m.err = msg.Err
		return m, nil

	case RefreshMsg:
		m.loading = true
		return m, m.loadTicket()

	case sseConnectedMsg:
		m.eventCh = msg.ch
		m.cancelEvents = msg.cancel
		return m, m.waitForEvent()

	case EventMsg:
		m.loading = true
		return m, tea.Batch(m.loadTicket(), m.waitForEvent())

	case DiffExecutedMsg:
		m.executingDiff = false
		return m, nil

	case DiffErrorMsg:
		m.executingDiff = false
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
	if m.showDetailModal {
		return m.handleDetailModalKey(msg)
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

	// If loading, killing, approving, spawning, rejecting, or executing diff, don't process other keys.
	if m.loading || m.killing || m.approving || m.spawning || m.rejecting || m.executingDiff {
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

	// Tab/Shift+Tab/[/] to switch row focus.
	if isKey(msg, KeyTab, KeyShiftTab, KeyLeftBracket, KeyRightBracket) {
		m.focusedRow = 1 - m.focusedRow // toggle 0↔1
		m.updateRowSizes()
		return m, nil
	}

	// Refresh.
	if isKey(msg, KeyRefresh) {
		m.loading = true
		return m, m.loadTicket()
	}

	// Comment list navigation (Row 2 focused).
	if m.focusedRow == 1 {
		return m.handleCommentListKey(msg)
	}

	// Body viewport navigation (Row 1 focused).

	// Kill session.
	if isKey(msg, KeyKillSession) {
		if m.hasActiveSession() {
			m.showKillModal = true
		}
		return m, nil
	}

	// Approve session (guarded: don't trigger on 'a' when 'g' is pending for 'ga').
	if !m.pendingG && isKey(msg, KeyApprove) {
		if m.hasActiveSession() && m.hasReviewRequests() {
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

// handleCommentListKey handles keyboard input when Row 2 (comment list) is focused.
func (m Model) handleCommentListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	itemCount := 0
	if m.ticket != nil {
		itemCount = len(m.ticket.Comments)
	}

	// Open detail modal.
	if isKey(msg, KeyO, KeyEnter) {
		if itemCount > 0 {
			m.openDetailModal()
		}
		return m, nil
	}

	// Cursor movement.
	if isKey(msg, KeyDown, KeyJ) {
		if itemCount > 0 {
			m.commentCursor = min(m.commentCursor+1, itemCount-1)
		}
		return m, nil
	}
	if isKey(msg, KeyUp, KeyK) {
		if itemCount > 0 {
			m.commentCursor = max(m.commentCursor-1, 0)
		}
		return m, nil
	}

	// Handle 'G' - jump to last item.
	if isKey(msg, KeyShiftG) {
		m.pendingG = false
		if itemCount > 0 {
			m.commentCursor = itemCount - 1
		}
		return m, nil
	}

	// Handle 'g' key for 'gg' sequence.
	if isKey(msg, KeyG) {
		if m.pendingG {
			m.pendingG = false
			m.commentCursor = 0
		} else {
			m.pendingG = true
		}
		return m, nil
	}

	// Handle 'ga' - focus architect window.
	if m.pendingG && isKey(msg, KeyApprove) {
		m.pendingG = false
		return m, m.focusArchitect()
	}

	// Clear pending g on any other key.
	m.pendingG = false

	// Global shortcuts available from comment list.
	if isKey(msg, KeyKillSession) {
		if m.hasActiveSession() {
			m.showKillModal = true
		}
		return m, nil
	}
	if isKey(msg, KeySpawn) {
		if m.canSpawn() {
			m.spawning = true
			return m, m.spawnSession()
		}
		return m, nil
	}
	if isKey(msg, KeyApprove) {
		if m.hasActiveSession() && m.hasReviewRequests() {
			m.approving = true
			return m, m.approveSession()
		}
		return m, nil
	}

	return m, nil
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

// handleDetailModalKey handles keyboard input when the detail modal is open.
func (m Model) handleDetailModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Close modal.
	if isKey(msg, KeyEscape, KeyQuit) {
		m.showDetailModal = false
		return m, nil
	}

	// Scroll modal content.
	if isKey(msg, KeyDown, KeyJ) {
		m.modalViewport.ScrollDown(1)
		return m, nil
	}
	if isKey(msg, KeyUp, KeyK) {
		m.modalViewport.ScrollUp(1)
		return m, nil
	}

	// Review-specific actions.
	if m.ticket != nil && m.modalCommentIdx < len(m.ticket.Comments) &&
		m.ticket.Comments[m.modalCommentIdx].Type == "review_requested" {
		if isKey(msg, KeyApprove) {
			if m.hasActiveSession() && m.hasReviewRequests() {
				m.showDetailModal = false
				m.approving = true
				return m, m.approveSession()
			}
			return m, nil
		}
		if isKey(msg, KeyKillSession) { // 'x' for reject
			m.showDetailModal = false
			m.rejecting = true
			return m, m.rejectSession()
		}
		// 'd' for diff - execute git_diff action if available
		if isKey(msg, KeyDiff) {
			comment := m.ticket.Comments[m.modalCommentIdx]
			if comment.Action != nil && comment.Action.Type == "git_diff" {
				m.showDetailModal = false
				m.executingDiff = true
				return m, m.executeDiffAction(comment.ID)
			}
			return m, nil
		}
	}

	return m, nil
}

// hasActiveSession returns true if there's an active (not ended) session.
func (m Model) hasActiveSession() bool {
	return m.ticket != nil && m.ticket.Session != nil && m.ticket.Session.EndedAt == nil
}

// hasReviewRequests returns true if there are review_requested comments.
func (m Model) hasReviewRequests() bool {
	if m.ticket == nil {
		return false
	}
	for _, c := range m.ticket.Comments {
		if c.Type == "review_requested" {
			return true
		}
	}
	return false
}

// killSession returns a command to kill the current session.
func (m Model) killSession() tea.Cmd {
	return func() tea.Msg {
		if m.ticket == nil || m.ticket.Session == nil {
			return SessionKillErrorMsg{Err: fmt.Errorf("no session to kill")}
		}
		err := m.client.KillSession(m.ticket.Session.ID)
		if err != nil {
			return SessionKillErrorMsg{Err: err}
		}
		return SessionKilledMsg{}
	}
}

// approveSession returns a command to approve the current session.
func (m Model) approveSession() tea.Cmd {
	return func() tea.Msg {
		if m.ticket == nil || m.ticket.Session == nil {
			return ApproveErrorMsg{Err: fmt.Errorf("no session to approve")}
		}
		err := m.client.ApproveSession(m.ticket.Session.ID)
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

// rejectSession returns a command to reject the current session review.
func (m Model) rejectSession() tea.Cmd {
	return func() tea.Msg {
		if m.ticket == nil {
			return RejectErrorMsg{Err: fmt.Errorf("no ticket")}
		}
		_, err := m.client.AddComment(m.ticket.ID, "comment", "Review rejected from TUI")
		if err != nil {
			return RejectErrorMsg{Err: err}
		}
		return SessionRejectedMsg{}
	}
}

// executeDiffAction returns a command to execute a git_diff action on a comment.
func (m Model) executeDiffAction(commentID string) tea.Cmd {
	return func() tea.Msg {
		if m.ticket == nil {
			return DiffErrorMsg{Err: fmt.Errorf("no ticket")}
		}
		err := m.client.ExecuteCommentAction(m.ticket.ID, commentID)
		if err != nil {
			return DiffErrorMsg{Err: err}
		}
		return DiffExecutedMsg{}
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
	if isKey(msg, KeyCancel, KeyEscape) {
		m.showOrphanModal = false
		return m, nil
	}
	return m, nil
}

// openDetailModal opens the detail modal for the currently selected comment.
func (m *Model) openDetailModal() {
	m.showDetailModal = true
	m.modalCommentIdx = m.commentCursor

	// Size: ~60% width, ~70% height minus chrome (border + padding + header + separator + help).
	modalInnerWidth := max(m.width*60/100-6, 20)  // 6 = border(2) + padding(4)
	modalInnerHeight := max(m.height*70/100-7, 5) // 7 = border(2) + padding(2) + header(1) + separator(1) + help(1)

	m.modalViewport = viewport.New(modalInnerWidth, modalInnerHeight)
	m.modalViewport.SetContent(m.renderModalContent(modalInnerWidth))
}

// renderDetailModal renders the centered detail modal overlay.
func (m Model) renderDetailModal() string {
	modalInnerWidth := max(m.width*60/100-6, 20)

	header := m.renderModalHeader()
	separator := modalSeparatorStyle.Render(strings.Repeat("─", modalInnerWidth))
	body := m.modalViewport.View()

	isReview := false
	hasAction := false
	if m.ticket != nil && m.modalCommentIdx < len(m.ticket.Comments) {
		comment := m.ticket.Comments[m.modalCommentIdx]
		isReview = comment.Type == "review_requested"
		hasAction = comment.Action != nil && comment.Action.Type == "git_diff"
	}
	help := modalHelpStyle.Render(modalHelpText(isReview, hasAction))

	content := header + "\n" + separator + "\n" + body + "\n" + help

	modal := modalStyle.Render(content)

	return lipgloss.Place(m.width, m.height-1, lipgloss.Center, lipgloss.Center, modal)
}

// renderModalHeader renders the header line for the detail modal.
func (m Model) renderModalHeader() string {
	if m.ticket == nil || m.modalCommentIdx >= len(m.ticket.Comments) {
		return modalHeaderStyle.Render("Comment")
	}
	comment := m.ticket.Comments[m.modalCommentIdx]
	badge := commentBadge(comment.Type)
	date := attributeLabelStyle.Render(formatTimeAgo(comment.CreatedAt))
	return badge + "  " + date
}

// renderModalContent renders the scrollable content for the detail modal.
func (m Model) renderModalContent(width int) string {
	if m.ticket == nil || m.modalCommentIdx >= len(m.ticket.Comments) {
		return ""
	}
	comment := m.ticket.Comments[m.modalCommentIdx]

	var b strings.Builder

	// Show repo path for review requests.
	if comment.Type == "review_requested" {
		repoPath := extractRepoPath(comment)
		if repoPath != "" && repoPath != "." {
			b.WriteString(modalRepoStyle.Render("Repo: " + repoPath))
			b.WriteString("\n\n")
		}
	}

	// Render content as markdown with modal-appropriate width.
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if renderer != nil {
		rendered, err := renderer.Render(comment.Content)
		if err == nil {
			b.WriteString(strings.TrimSpace(rendered))
			return b.String()
		}
	}
	b.WriteString(comment.Content)
	return b.String()
}

// extractRepoPath extracts repo_path from a review comment's Action.Args.
func extractRepoPath(comment sdk.CommentResponse) string {
	if comment.Action == nil {
		return ""
	}
	argsMap, ok := comment.Action.Args.(map[string]any)
	if !ok {
		return ""
	}
	repoPath, _ := argsMap["repo_path"].(string)
	return repoPath
}

// renderOrphanModal renders the orphaned session modal.
func (m Model) renderOrphanModal() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(warningStyle.Render("Orphaned session detected"))
	b.WriteString("\n\n")
	b.WriteString("The tmux window for this session was closed.\n\n")
	b.WriteString("[r]esume  [f]resh  [c]ancel")

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

	// Handle rejecting state.
	if m.rejecting {
		b.WriteString(loadingStyle.Render("Rejecting session..."))
		return b.String()
	}

	// Handle executing diff state.
	if m.executingDiff {
		b.WriteString(loadingStyle.Render("Opening diff..."))
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

	// Detail modal (overlay).
	if m.showDetailModal {
		return m.renderDetailModal()
	}

	// Row 1 (body + attributes).
	row1H, row2H := m.rowHeights()
	b.WriteString(m.renderRow1(row1H))

	// Row separator.
	b.WriteString("\n")
	b.WriteString(m.renderRowSeparator())

	// Row 2 (comment list).
	b.WriteString("\n")
	b.WriteString(m.renderCommentList(m.width, row2H))

	// Help bar.
	b.WriteString("\n")
	b.WriteString(helpBarStyle.Render(helpText(
		int(m.bodyViewport.ScrollPercent()*100),
		m.hasActiveSession(), m.hasReviewRequests(), m.canSpawn(),
		m.embedded, m.focusedRow,
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

	// ID + Title + Status badge.
	id := ticketIDStyle.Render(m.ticket.ID[:8])
	title := titleStyle.Render(m.ticket.Title)
	status := statusStyle(m.ticket.Status).Render(m.ticket.Status)

	left := id + "  " + title
	right := status

	padding := max(m.width-lipgloss.Width(left)-lipgloss.Width(right)-2, 1)
	return left + strings.Repeat(" ", padding) + right
}

// rowHeights computes the vertical split between Row 1 and Row 2.
func (m Model) rowHeights() (row1H, row2H int) {
	// available = height - 4 (header with padding + row separator + help bar)
	available := max(m.height-4, 2)
	if m.focusedRow == 0 {
		row1H = available * 70 / 100
	} else {
		row1H = available * 30 / 100
	}
	row1H = max(row1H, 1)
	row2H = max(available-row1H, 1)
	return
}

// renderRow1 renders Row 1: body viewport + attributes panel (wide) or body only (narrow).
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
	b.WriteString(attributeValueStyle.Render(m.ticket.Dates.Created.Format("Jan 02, 15:04")))

	b.WriteString("\n")
	b.WriteString(attributeLabelStyle.Render("Updated  "))
	b.WriteString(attributeValueStyle.Render(m.ticket.Dates.Updated.Format("Jan 02, 15:04")))

	if m.ticket.Dates.Progress != nil {
		b.WriteString("\n")
		b.WriteString(attributeLabelStyle.Render("Progress "))
		b.WriteString(attributeValueStyle.Render(m.ticket.Dates.Progress.Format("Jan 02, 15:04")))
	}

	if m.ticket.Dates.Reviewed != nil {
		b.WriteString("\n")
		b.WriteString(attributeLabelStyle.Render("Reviewed "))
		b.WriteString(attributeValueStyle.Render(m.ticket.Dates.Reviewed.Format("Jan 02, 15:04")))
	}

	if m.ticket.Dates.Done != nil {
		b.WriteString("\n")
		b.WriteString(attributeLabelStyle.Render("Done     "))
		b.WriteString(attributeValueStyle.Render(m.ticket.Dates.Done.Format("Jan 02, 15:04")))
	}

	// SESSION section.
	if m.ticket.Session != nil {
		session := m.ticket.Session
		b.WriteString("\n\n")
		b.WriteString(attributeHeaderStyle.Render("SESSION"))
		b.WriteString("\n")

		b.WriteString(attributeLabelStyle.Render("Agent    "))
		b.WriteString(attributeValueStyle.Render(session.Agent))

		b.WriteString("\n")
		b.WriteString(attributeLabelStyle.Render("Status   "))
		if session.EndedAt == nil {
			b.WriteString(statusStyle("progress").Render("ACTIVE"))
		} else {
			b.WriteString(statusStyle("done").Render("ENDED"))
		}

		if session.CurrentStatus != nil && session.CurrentStatus.Tool != nil {
			b.WriteString("\n")
			b.WriteString(attributeLabelStyle.Render("Tool     "))
			b.WriteString(attributeValueStyle.Render(*session.CurrentStatus.Tool))
		}

		b.WriteString("\n")
		b.WriteString(attributeLabelStyle.Render("Window   "))
		b.WriteString(attributeValueStyle.Render(session.TmuxWindow))

		b.WriteString("\n")
		b.WriteString(attributeLabelStyle.Render("Started  "))
		b.WriteString(attributeValueStyle.Render(session.StartedAt.Format("Jan 02, 15:04")))

		if session.EndedAt != nil {
			b.WriteString("\n")
			b.WriteString(attributeLabelStyle.Render("Ended    "))
			b.WriteString(attributeValueStyle.Render(session.EndedAt.Format("Jan 02, 15:04")))
		}
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		PaddingLeft(1).
		Render(b.String())
}

// renderCommentList renders the unified comment list for Row 2.
func (m Model) renderCommentList(width, height int) string {
	if m.ticket == nil || len(m.ticket.Comments) == 0 {
		content := attributeLabelStyle.Render("No comments")
		if m.focusedRow == 1 {
			return row2FocusedStyle.Width(width).Height(height).Render(content)
		}
		return row2Style.Width(width).Height(height).Render(content)
	}

	// Header line takes 1 row.
	headerLine := attributeHeaderStyle.Render(fmt.Sprintf("COMMENTS (%d)", len(m.ticket.Comments)))
	availableHeight := max(height-1, 1) // 1 for header

	// Determine visible range.
	start, end := m.commentVisibleRange(availableHeight)

	// Content width: account for border/padding (~2 chars).
	contentWidth := max(width-2, 10)

	var b strings.Builder
	b.WriteString(headerLine)

	for i := start; i < end; i++ {
		b.WriteString("\n")
		selected := m.focusedRow == 1 && i == m.commentCursor
		b.WriteString(m.renderCommentRow(m.ticket.Comments[i], contentWidth, selected))
		// Add padding between rows (except after last).
		if i < end-1 {
			for j := 0; j < CommentRowPadding; j++ {
				b.WriteString("\n")
			}
		}
	}

	content := b.String()
	if m.focusedRow == 1 {
		return row2FocusedStyle.Width(width).Height(height).Render(content)
	}
	return row2Style.Width(width).Height(height).Render(content)
}

// renderCommentRow renders a single comment as a multi-line row (4 lines: header + 3 preview lines).
func (m Model) renderCommentRow(comment sdk.CommentResponse, width int, selected bool) string {
	badge := commentBadge(comment.Type)
	timeAgo := formatTimeAgo(comment.CreatedAt)

	// Build header line: [badge]  repo-prefix (if review)  ────  time-ago
	var headerParts []string
	headerParts = append(headerParts, badge)

	// For review requests, show repo name.
	if comment.Type == "review_requested" {
		repoPath := extractRepoPath(comment)
		repo := filepath.Base(repoPath)
		if repo != "" && repo != "." && repo != "/" {
			headerParts = append(headerParts, attributeLabelStyle.Render(repo))
		}
	}

	headerLeft := strings.Join(headerParts, "  ")
	headerLeftWidth := lipgloss.Width(headerLeft)
	timeWidth := lipgloss.Width(timeAgo)

	// Fill with separator dashes between header and time.
	separatorWidth := max(width-headerLeftWidth-timeWidth-4, 1) // 4 = 2 spaces on each side
	separator := attributeLabelStyle.Render(strings.Repeat("─", separatorWidth))

	headerLine := headerLeft + "  " + separator + "  " + attributeLabelStyle.Render(timeAgo)

	// Render markdown content and get up to 3 lines.
	previewLines := m.renderCommentPreview(comment.Content, width, 3)

	// Build the full row (4 lines total).
	var lines []string
	lines = append(lines, headerLine)
	lines = append(lines, previewLines...)
	// Pad to exactly CommentRowLines lines.
	for len(lines) < CommentRowLines {
		lines = append(lines, "")
	}

	// Apply background to entire block if selected.
	if selected {
		return m.applyBackgroundToBlock(lines, width, commentSelectedStyle.GetBackground())
	}

	return strings.Join(lines, "\n")
}

// renderCommentPreview renders markdown and returns up to maxLines lines.
func (m Model) renderCommentPreview(content string, width, maxLines int) []string {
	if content == "" {
		return []string{attributeLabelStyle.Render("(empty)")}
	}

	// Create a renderer for the preview width.
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil || renderer == nil {
		// Fallback to plain text.
		return m.plainTextPreview(content, width, maxLines)
	}

	rendered, err := renderer.Render(content)
	if err != nil {
		return m.plainTextPreview(content, width, maxLines)
	}

	// Split rendered output into lines and take up to maxLines.
	rendered = strings.TrimSpace(rendered)
	allLines := strings.Split(rendered, "\n")

	var result []string
	for _, line := range allLines {
		if len(result) >= maxLines {
			break
		}
		// Skip empty lines at the start.
		if len(result) == 0 && strings.TrimSpace(line) == "" {
			continue
		}
		// Truncate line to width, preserving ANSI codes.
		truncated := ansi.Truncate(line, width, "…")
		result = append(result, truncated)
	}

	if len(result) == 0 {
		return []string{attributeLabelStyle.Render("(empty)")}
	}

	return result
}

// plainTextPreview returns plain text lines as a fallback.
func (m Model) plainTextPreview(content string, width, maxLines int) []string {
	lines := strings.Split(content, "\n")
	var result []string
	for _, line := range lines {
		if len(result) >= maxLines {
			break
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" && len(result) == 0 {
			continue // Skip leading empty lines.
		}
		// Strip markdown syntax.
		trimmed = strings.TrimLeft(trimmed, "#*-> ")
		trimmed = strings.TrimSpace(trimmed)
		if trimmed != "" {
			result = append(result, truncateToWidth(trimmed, width))
		}
	}
	if len(result) == 0 {
		return []string{attributeLabelStyle.Render("(empty)")}
	}
	return result
}

// applyBackgroundToBlock applies a background color to all lines of a block.
func (m Model) applyBackgroundToBlock(lines []string, width int, bgColor lipgloss.TerminalColor) string {
	style := lipgloss.NewStyle().Background(bgColor).Width(width)
	var result []string
	for _, line := range lines {
		// Pad line to full width before applying background.
		lineWidth := lipgloss.Width(line)
		if lineWidth < width {
			line += strings.Repeat(" ", width-lineWidth)
		}
		result = append(result, style.Render(line))
	}
	return strings.Join(result, "\n")
}

// commentBadge returns a styled badge string for a comment type.
func commentBadge(commentType string) string {
	badgeText := commentType
	switch commentType {
	case "review_requested":
		badgeText = "review"
	}
	return commentTypeStyle(commentType).Render("[" + badgeText + "]")
}

// truncateToWidth truncates a string to fit within maxWidth, appending "…" if truncated.
func truncateToWidth(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	// Truncate rune by rune.
	var result strings.Builder
	currentWidth := 0
	for _, r := range s {
		charWidth := lipgloss.Width(string(r))
		if currentWidth+charWidth+1 > maxWidth { // +1 for "…"
			result.WriteString("…")
			return result.String()
		}
		result.WriteRune(r)
		currentWidth += charWidth
	}
	return result.String()
}

// commentVisibleRange computes the visible window of comments keeping the cursor visible.
func (m Model) commentVisibleRange(visibleHeight int) (start, end int) {
	total := len(m.ticket.Comments)
	if total == 0 {
		return 0, 0
	}

	// Each comment row takes CommentRowLines lines, with CommentRowPadding between rows.
	// For n comments, total lines = n * CommentRowLines + (n-1) * CommentRowPadding
	// Simplified: rowHeight per comment = CommentRowLines + CommentRowPadding (except last)
	rowHeight := CommentRowLines + CommentRowPadding // 5 lines per comment
	visibleCount := max(visibleHeight/rowHeight, 1)

	start = 0
	end = min(visibleCount, total)

	if m.commentCursor >= end {
		end = min(m.commentCursor+1, total)
		start = max(end-visibleCount, 0)
	}
	if m.commentCursor < start {
		start = m.commentCursor
		end = min(start+visibleCount, total)
	}

	return start, end
}

// renderRowSeparator renders a thin horizontal line between rows.
func (m Model) renderRowSeparator() string {
	return rowSeparatorStyle.Render(strings.Repeat("─", m.width))
}

// updateRowSizes resizes the bodyViewport after a focus change.
func (m *Model) updateRowSizes() {
	if !m.ready {
		return
	}
	row1H, _ := m.rowHeights()
	vpWidth := m.width
	if m.width >= wideLayoutMinWidth {
		vpWidth = m.width * 70 / 100
	}
	m.bodyViewport.Width = vpWidth
	m.bodyViewport.Height = row1H
	if m.ticket != nil {
		m.bodyViewport.SetContent(m.renderBodyContent())
	}
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

	// Narrow mode: inline dates + body + session.
	var b strings.Builder

	// Inline dates.
	b.WriteString(labelStyle.Render("Created: "))
	b.WriteString(valueStyle.Render(m.ticket.Dates.Created.Format("Jan 02, 2006 15:04")))
	b.WriteString("  ")
	b.WriteString(labelStyle.Render("Updated: "))
	b.WriteString(valueStyle.Render(m.ticket.Dates.Updated.Format("Jan 02, 2006 15:04")))

	// Body.
	if m.ticket.Body != "" {
		b.WriteString("\n\n")
		b.WriteString(m.renderMarkdown(m.ticket.Body))
	}

	// Session.
	if m.ticket.Session != nil {
		session := m.ticket.Session
		b.WriteString("\n\n")
		b.WriteString(sectionHeaderStyle.Render("─── Session ───"))
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("Agent: "))
		b.WriteString(valueStyle.Render(session.Agent))
		b.WriteString("  ")
		b.WriteString(labelStyle.Render("Status: "))
		if session.EndedAt == nil {
			b.WriteString(statusStyle("progress").Render("ACTIVE"))
		} else {
			b.WriteString(statusStyle("done").Render("ENDED"))
		}
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

// loadTicket returns a command to load the ticket.
func (m Model) loadTicket() tea.Cmd {
	return func() tea.Msg {
		ticket, err := m.client.FindTicketByID(m.ticketID)
		if err != nil {
			return TicketErrorMsg{Err: err}
		}
		return TicketLoadedMsg{Ticket: ticket}
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
		return nil // channel closed
	}
}

// formatTimeAgo formats a time as a human-readable relative string.
func formatTimeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d min ago", mins)
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}
