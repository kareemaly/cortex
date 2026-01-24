package ticket

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/kareemaly/cortex/internal/cli/sdk"
)

// Model is the main Bubbletea model for the ticket detail view.
type Model struct {
	client        *sdk.Client
	ticketID      string
	ticket        *sdk.TicketResponse
	viewport      viewport.Model
	width         int
	height        int
	ready         bool
	loading       bool
	err           error
	showKillModal bool
	killing       bool
	embedded      bool // if true, send CloseDetailMsg instead of tea.Quit
	pendingG      bool // tracking 'g' key for 'gg' sequence
	mdRenderer    *glamour.TermRenderer
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

// CloseDetailMsg is sent when user wants to close the detail view.
type CloseDetailMsg struct{}

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

// Init initializes the model and starts loading the ticket.
func (m Model) Init() tea.Cmd {
	return m.loadTicket()
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Header (1 line) + blank (1) + help bar (1) = 3 lines overhead.
		viewportHeight := max(m.height-3, 1)

		// Update renderer width.
		renderer, _ := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(m.width),
		)
		m.mdRenderer = renderer

		if !m.ready {
			m.viewport = viewport.New(m.width, viewportHeight)
			m.viewport.YPosition = 2 // Below header.
			m.ready = true
			if m.ticket != nil {
				m.viewport.SetContent(m.renderContent())
			}
		} else {
			m.viewport.Width = m.width
			m.viewport.Height = viewportHeight
			if m.ticket != nil {
				m.viewport.SetContent(m.renderContent())
			}
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case TicketLoadedMsg:
		m.loading = false
		m.err = nil
		m.ticket = msg.Ticket
		if m.ready {
			m.viewport.SetContent(m.renderContent())
			m.viewport.GotoTop()
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
	}

	// Handle viewport scroll messages.
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// handleKeyMsg handles keyboard input.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Modal takes priority when visible.
	if m.showKillModal {
		return m.handleKillModalKey(msg)
	}

	// Quit or close.
	if isKey(msg, KeyQuit, KeyCtrlC) {
		if m.embedded {
			return m, func() tea.Msg { return CloseDetailMsg{} }
		}
		return m, tea.Quit
	}

	// Handle Escape for embedded mode.
	if m.embedded && isKey(msg, KeyEscape) {
		return m, func() tea.Msg { return CloseDetailMsg{} }
	}

	// If loading or killing, don't process other keys.
	if m.loading || m.killing {
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

	// Handle 'G' - jump to bottom.
	if isKey(msg, KeyShiftG) {
		m.pendingG = false
		m.viewport.GotoBottom()
		return m, nil
	}

	// Handle 'g' key for 'gg' sequence.
	if isKey(msg, KeyG) {
		if m.pendingG {
			// Second 'g' - jump to top.
			m.pendingG = false
			m.viewport.GotoTop()
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
		m.viewport.HalfPageUp()
		return m, nil
	}
	if isKey(msg, KeyCtrlD) {
		m.viewport.HalfPageDown()
		return m, nil
	}

	// Scroll navigation.
	if isKey(msg, KeyUp, KeyK) {
		m.viewport.ScrollUp(1)
		return m, nil
	}
	if isKey(msg, KeyDown, KeyJ) {
		m.viewport.ScrollDown(1)
		return m, nil
	}
	if isKey(msg, KeyPgUp) {
		m.viewport.PageUp()
		return m, nil
	}
	if isKey(msg, KeyPgDown) {
		m.viewport.PageDown()
		return m, nil
	}
	if isKey(msg, KeyHome) {
		m.viewport.GotoTop()
		return m, nil
	}
	if isKey(msg, KeyEnd) {
		m.viewport.GotoBottom()
		return m, nil
	}

	// Pass to viewport for mouse scroll, etc.
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
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

// hasActiveSession returns true if there's an active (not ended) session.
func (m Model) hasActiveSession() bool {
	return m.ticket != nil && m.ticket.Session != nil && m.ticket.Session.EndedAt == nil
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

	// Kill confirmation modal.
	if m.showKillModal {
		b.WriteString(m.renderKillModal())
		return b.String()
	}

	// Scrollable content.
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// Help bar.
	b.WriteString(helpBarStyle.Render(helpText(int(m.viewport.ScrollPercent()*100), m.hasActiveSession(), m.embedded)))

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

// renderContent renders the scrollable content for the viewport.
func (m Model) renderContent() string {
	if m.ticket == nil {
		return ""
	}

	var b strings.Builder

	// Dates section.
	b.WriteString(m.renderDates())
	b.WriteString("\n")

	// Body section.
	if m.ticket.Body != "" {
		b.WriteString(m.renderSection("Description", m.ticket.Body))
		b.WriteString("\n")
	}

	// Session section.
	if m.ticket.Session != nil {
		b.WriteString(m.renderSession())
		b.WriteString("\n")
	}

	// Comments section.
	if len(m.ticket.Comments) > 0 {
		b.WriteString(m.renderComments())
	}

	return b.String()
}

// renderDates renders the dates section.
func (m Model) renderDates() string {
	var b strings.Builder

	b.WriteString(labelStyle.Render("Created:   "))
	b.WriteString(valueStyle.Render(m.ticket.Dates.Created.Format("Jan 02, 2006 15:04")))
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("Updated:   "))
	b.WriteString(valueStyle.Render(m.ticket.Dates.Updated.Format("Jan 02, 2006 15:04")))

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

// renderSection renders a labeled section with content.
func (m Model) renderSection(title, content string) string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(sectionHeaderStyle.Render("─── " + title + " ───"))
	b.WriteString("\n")
	b.WriteString(m.renderMarkdown(content))

	return b.String()
}

// renderSession renders the session section.
func (m Model) renderSession() string {
	session := m.ticket.Session
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(sectionHeaderStyle.Render("─── Session ───"))
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("ID:        "))
	b.WriteString(valueStyle.Render(session.ID[:8]))
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("Agent:     "))
	b.WriteString(valueStyle.Render(session.Agent))
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("Status:    "))
	if session.EndedAt == nil {
		b.WriteString(statusStyle("progress").Render("ACTIVE"))
	} else {
		b.WriteString(statusStyle("done").Render("ENDED"))
	}
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("Started:   "))
	b.WriteString(valueStyle.Render(session.StartedAt.Format("Jan 02, 2006 15:04")))

	if session.EndedAt != nil {
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("Ended:     "))
		b.WriteString(valueStyle.Render(session.EndedAt.Format("Jan 02, 2006 15:04")))
	}

	// Current status if present.
	if session.CurrentStatus != nil {
		b.WriteString("\n\n")
		b.WriteString(labelStyle.Render("Current:   "))
		b.WriteString(valueStyle.Render(session.CurrentStatus.Status))
		if session.CurrentStatus.Tool != nil {
			b.WriteString(" (")
			b.WriteString(valueStyle.Render(*session.CurrentStatus.Tool))
			b.WriteString(")")
		}
		if session.CurrentStatus.Work != nil && *session.CurrentStatus.Work != "" {
			b.WriteString("\n")
			b.WriteString(labelStyle.Render("           "))
			b.WriteString(valueStyle.Render(*session.CurrentStatus.Work))
		}
	}

	return b.String()
}

// renderComments renders the comments section.
func (m Model) renderComments() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(sectionHeaderStyle.Render(fmt.Sprintf("─── Comments (%d) ───", len(m.ticket.Comments))))
	b.WriteString("\n")

	for i, comment := range m.ticket.Comments {
		// Comment type badge and date.
		typeStyle := commentTypeStyle(comment.Type)
		badge := typeStyle.Render("[" + comment.Type + "]")
		date := labelStyle.Render(comment.CreatedAt.Format("Jan 02 15:04"))

		b.WriteString(badge)
		b.WriteString(strings.Repeat(" ", max(15-len(comment.Type), 1)))
		b.WriteString(date)
		b.WriteString("\n")

		// Comment content rendered as markdown.
		b.WriteString(m.renderMarkdown(comment.Content))
		b.WriteString("\n")

		// Add spacing between comments, but not after the last one.
		if i < len(m.ticket.Comments)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
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
