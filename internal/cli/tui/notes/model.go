package notes

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/cli/tui/tuilog"
)

// Input mode for the text input field.
type inputMode int

const (
	inputNone     inputMode = iota
	inputNewText            // Creating: entering note text
	inputNewDate            // Creating: entering due date
	inputEditText           // Editing note text
	inputEditDate           // Editing due date
)

// SSE reconnection constants.
const (
	sseInitialBackoff = 2 * time.Second
	sseMaxBackoff     = 30 * time.Second
	pollInterval      = 60 * time.Second
)

// Model is the Bubbletea model for the notes view.
type Model struct {
	client *sdk.Client
	notes  []sdk.NoteResponse
	cursor int

	listVP viewport.Model

	input           textinput.Model
	mode            inputMode
	pendingNoteText string // stash text during new-note two-step flow
	editNoteID      string // ID of note being edited

	showDeleteModal bool
	deleteNoteID    string

	width, height int
	ready         bool
	loading       bool
	err           error
	pendingG      bool

	// SSE subscription state.
	eventCh      <-chan sdk.Event
	cancelEvents context.CancelFunc
	sseBackoff   time.Duration
	sseConnected bool

	// Log viewer state.
	logBuf        *tuilog.Buffer
	logViewer     tuilog.Viewer
	showLogViewer bool

	statusMsg     string
	statusIsError bool
}

// Message types for async operations.

// NotesLoadedMsg is sent when notes are successfully fetched.
type NotesLoadedMsg struct {
	Notes []sdk.NoteResponse
}

// NotesErrorMsg is sent when fetching notes fails.
type NotesErrorMsg struct {
	Err error
}

// NoteCreatedMsg is sent when a note is created.
type NoteCreatedMsg struct {
	Err error
}

// NoteUpdatedMsg is sent when a note is updated.
type NoteUpdatedMsg struct {
	Err error
}

// NoteDeletedMsg is sent when a note is deleted.
type NoteDeletedMsg struct {
	Err error
}

// ClearStatusMsg clears the status message after a delay.
type ClearStatusMsg struct{}

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

// New creates a new notes model.
func New(client *sdk.Client, logBuf *tuilog.Buffer) Model {
	ti := textinput.New()
	ti.CharLimit = 500
	return Model{
		client:    client,
		loading:   true,
		input:     ti,
		logBuf:    logBuf,
		logViewer: tuilog.NewViewer(logBuf),
	}
}

// Init starts loading notes and subscribing to events.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadNotes(), m.subscribeEvents(), m.startPollTicker())
}

// InputActive returns true when the notes view is capturing input (text input or delete modal).
func (m Model) InputActive() bool {
	return m.mode != inputNone || m.showDeleteModal
}

// Update handles messages.
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

	case NotesLoadedMsg:
		m.loading = false
		m.err = nil
		m.notes = sortNotes(msg.Notes)
		if m.cursor >= len(m.notes) {
			m.cursor = max(len(m.notes)-1, 0)
		}
		m.logBuf.Debug("api", "notes loaded")
		return m, nil

	case NotesErrorMsg:
		m.loading = false
		m.err = msg.Err
		m.logBuf.Errorf("api", "failed to load notes: %s", msg.Err)
		return m, nil

	case NoteCreatedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Create failed: %s", msg.Err)
			m.statusIsError = true
		} else {
			m.statusMsg = "Note created"
			m.statusIsError = false
		}
		return m, tea.Batch(m.loadNotes(), m.clearStatusAfterDelay())

	case NoteUpdatedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Update failed: %s", msg.Err)
			m.statusIsError = true
		} else {
			m.statusMsg = "Note updated"
			m.statusIsError = false
		}
		return m, tea.Batch(m.loadNotes(), m.clearStatusAfterDelay())

	case NoteDeletedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Delete failed: %s", msg.Err)
			m.statusIsError = true
		} else {
			m.statusMsg = "Note deleted"
			m.statusIsError = false
		}
		return m, tea.Batch(m.loadNotes(), m.clearStatusAfterDelay())

	case ClearStatusMsg:
		m.statusMsg = ""
		m.statusIsError = false
		return m, nil

	case sseConnectedMsg:
		if m.cancelEvents != nil {
			m.cancelEvents()
		}
		m.eventCh = msg.ch
		m.cancelEvents = msg.cancel
		m.sseConnected = true
		m.sseBackoff = 0
		m.logBuf.Info("sse", "connected to event stream")
		return m, tea.Batch(m.loadNotes(), m.waitForEvent())

	case EventMsg:
		m.logBuf.Debug("sse", "event received")
		return m, tea.Batch(m.loadNotes(), m.waitForEvent())

	case sseDisconnectedMsg:
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
		return m, tea.Batch(m.loadNotes(), m.startPollTicker())
	}

	return m, nil
}

// handleKeyMsg handles keyboard input.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Quit.
	if isKey(msg, KeyCtrlC) {
		if m.cancelEvents != nil {
			m.cancelEvents()
		}
		return m, tea.Quit
	}

	// Input mode — route to input handler.
	if m.mode != inputNone {
		return m.handleInputKey(msg)
	}

	// Delete modal — route to modal handler.
	if m.showDeleteModal {
		return m.handleDeleteModalKey(msg)
	}

	// Toggle log viewer.
	if isKey(msg, KeyBang) {
		m.showLogViewer = !m.showLogViewer
		if m.showLogViewer {
			m.logViewer.SetSize(m.width, m.height)
			m.logViewer.Reset()
		}
		return m, nil
	}

	// Don't process other keys while loading or if there's an error (except refresh).
	if m.loading {
		return m, nil
	}
	if m.err != nil {
		if isKey(msg, KeyR) {
			m.loading = true
			m.err = nil
			return m, m.loadNotes()
		}
		if isKey(msg, KeyQuit) {
			if m.cancelEvents != nil {
				m.cancelEvents()
			}
			return m, tea.Quit
		}
		return m, nil
	}

	return m.handleNormalKey(msg)
}

// handleNormalKey handles keys in the normal (non-input, non-modal) state.
func (m Model) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Quit.
	if isKey(msg, KeyQuit) {
		if m.cancelEvents != nil {
			m.cancelEvents()
		}
		return m, tea.Quit
	}

	// Refresh.
	if isKey(msg, KeyR) {
		m.loading = true
		return m, m.loadNotes()
	}

	// Handle 'G' — jump to last.
	if isKey(msg, KeyShiftG) {
		m.pendingG = false
		if len(m.notes) > 0 {
			m.cursor = len(m.notes) - 1
		}
		return m, nil
	}

	// Handle 'g' key for 'gg' sequence.
	if isKey(msg, KeyG) {
		if m.pendingG {
			m.pendingG = false
			m.cursor = 0
		} else {
			m.pendingG = true
		}
		return m, nil
	}

	// Clear pending g on any other key.
	m.pendingG = false

	switch {
	case isKey(msg, KeyJ, KeyDown):
		if m.cursor < len(m.notes)-1 {
			m.cursor++
		}
	case isKey(msg, KeyK, KeyUp):
		if m.cursor > 0 {
			m.cursor--
		}
	case isKey(msg, KeyCtrlD):
		m.cursor = min(m.cursor+10, max(len(m.notes)-1, 0))
	case isKey(msg, KeyCtrlU):
		m.cursor = max(m.cursor-10, 0)

	case isKey(msg, KeyN):
		// New note — step 1: enter text.
		m.mode = inputNewText
		m.input.SetValue("")
		m.input.Placeholder = "Note text..."
		m.input.Focus()
		return m, nil

	case isKey(msg, KeyE):
		// Edit text of selected note.
		if m.cursor >= 0 && m.cursor < len(m.notes) {
			note := m.notes[m.cursor]
			m.mode = inputEditText
			m.editNoteID = note.ID
			m.input.SetValue(note.Text)
			m.input.Placeholder = "Note text..."
			m.input.Focus()
		}
		return m, nil

	case isKey(msg, KeyT):
		// Edit due date of selected note.
		if m.cursor >= 0 && m.cursor < len(m.notes) {
			note := m.notes[m.cursor]
			m.mode = inputEditDate
			m.editNoteID = note.ID
			if note.Due != nil {
				m.input.SetValue(note.Due.Format("2006-01-02"))
			} else {
				m.input.SetValue("")
			}
			m.input.Placeholder = "YYYY-MM-DD (empty to clear)"
			m.input.Focus()
		}
		return m, nil

	case isKey(msg, KeySpace):
		// Mark done — delete without confirmation.
		if m.cursor >= 0 && m.cursor < len(m.notes) {
			note := m.notes[m.cursor]
			return m, m.deleteNote(note.ID)
		}

	case isKey(msg, KeyD):
		// Delete with confirmation.
		if m.cursor >= 0 && m.cursor < len(m.notes) {
			m.showDeleteModal = true
			m.deleteNoteID = m.notes[m.cursor].ID
		}
		return m, nil
	}

	return m, nil
}

// handleInputKey handles keys when the text input is active.
func (m Model) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if isKey(msg, KeyEsc) {
		m.mode = inputNone
		m.input.Blur()
		m.pendingNoteText = ""
		m.editNoteID = ""
		return m, nil
	}

	if isKey(msg, KeyEnter) {
		value := strings.TrimSpace(m.input.Value())
		m.input.Blur()

		switch m.mode {
		case inputNewText:
			if value == "" {
				m.mode = inputNone
				return m, nil
			}
			m.pendingNoteText = value
			m.mode = inputNewDate
			m.input.SetValue("")
			m.input.Placeholder = "YYYY-MM-DD (enter to skip)"
			m.input.Focus()
			return m, nil

		case inputNewDate:
			m.mode = inputNone
			text := m.pendingNoteText
			m.pendingNoteText = ""
			var due *string
			if value != "" {
				due = &value
			}
			return m, m.createNote(text, due)

		case inputEditText:
			m.mode = inputNone
			id := m.editNoteID
			m.editNoteID = ""
			if value == "" {
				return m, nil
			}
			return m, m.updateNote(id, &value, nil)

		case inputEditDate:
			m.mode = inputNone
			id := m.editNoteID
			m.editNoteID = ""
			// Empty value means clear due date (send empty string).
			return m, m.updateNote(id, nil, &value)
		}

		m.mode = inputNone
		return m, nil
	}

	// All other keys go to the text input.
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// handleDeleteModalKey handles keys when the delete confirmation modal is shown.
func (m Model) handleDeleteModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if isKey(msg, KeyY) {
		m.showDeleteModal = false
		id := m.deleteNoteID
		m.deleteNoteID = ""
		return m, m.deleteNote(id)
	}
	// Any other key dismisses the modal.
	m.showDeleteModal = false
	m.deleteNoteID = ""
	return m, nil
}

// View renders the notes view.
func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	// Log viewer overlay.
	if m.showLogViewer {
		return m.logViewer.View()
	}

	var b strings.Builder

	// Error state.
	if m.err != nil {
		b.WriteString(errorStatusStyle.Render(fmt.Sprintf("Error: %s", m.err)))
		b.WriteString("\n\n")
		b.WriteString("Press [r] to retry or [q] to quit\n")
		if strings.Contains(m.err.Error(), "connect") {
			b.WriteString("\nIs the daemon running? Start it with: cortexd start\n")
		}
		return b.String()
	}

	// Loading state.
	if m.loading {
		b.WriteString(loadingStyle.Render("Loading notes..."))
		return b.String()
	}

	// Height: status bar (1) + help bar (1) + optional input bar (1) = overhead.
	inputBarHeight := 0
	if m.mode != inputNone {
		inputBarHeight = 1
	}
	contentHeight := max(m.height-2-inputBarHeight, 3)

	// Note list.
	if len(m.notes) == 0 {
		empty := emptyStyle.Padding(1, 2).Render("No notes. Press [n] to create one.")
		b.WriteString(lipgloss.NewStyle().Width(m.width).Height(contentHeight).Render(empty))
	} else {
		b.WriteString(m.renderNoteList(contentHeight))
	}
	b.WriteString("\n")

	// Input bar (when in input mode).
	if m.mode != inputNone {
		label := m.inputLabel()
		b.WriteString(inputLabelStyle.Render(label) + " " + m.input.View())
		b.WriteString("\n")
	}

	// Status bar or delete modal.
	if m.showDeleteModal {
		b.WriteString(deleteModalStyle.Render("Delete this note? [y]es / [n]o"))
		b.WriteString("\n")
	} else if m.statusMsg != "" {
		style := statusBarStyle
		if m.statusIsError {
			style = errorStatusStyle
		}
		b.WriteString(style.Render(m.statusMsg))
		b.WriteString("\n")
	} else {
		b.WriteString("\n")
	}

	// Help bar.
	count := statusBarStyle.Render(fmt.Sprintf("%d notes", len(m.notes)))
	help := count + "  " + helpBarStyle.Render(helpText())
	if m.mode != inputNone {
		help = helpBarStyle.Render(inputHelpText())
	}
	badge := m.logBadge()
	if badge != "" {
		help = help + "  " + badge
	}
	b.WriteString(help)

	return b.String()
}

// renderNoteList renders the scrollable note list using a viewport.
// Notes are word-wrapped across multiple lines with metadata on a dedicated line.
func (m Model) renderNoteList(height int) string {
	var content strings.Builder
	now := time.Now()

	prefix := "  "       // indent for continuation and metadata lines
	cursorPrefix := "▸ " // first line of selected note
	contentWidth := max(m.width-lipgloss.Width(prefix), 10)

	noteStartLines := make([]int, len(m.notes))
	currentLine := 0

	for i, note := range m.notes {
		noteStartLines[i] = currentLine

		// Separate notes with a blank line (after the first).
		if i > 0 {
			content.WriteString("\n")
			currentLine++
		}

		// Word-wrap note text to full available width.
		wrapped := wrapToWidth(note.Text, contentWidth)

		for j, wline := range wrapped {
			if j == 0 {
				// First line gets cursor indicator.
				if i == m.cursor {
					content.WriteString(cursorPrefix)
					content.WriteString(selectedNoteStyle.Render(wline))
				} else {
					content.WriteString(prefix)
					content.WriteString(wline)
				}
			} else {
				// Continuation lines indented to align with text.
				content.WriteString("\n")
				currentLine++
				if i == m.cursor {
					content.WriteString(prefix)
					content.WriteString(selectedNoteStyle.Render(wline))
				} else {
					content.WriteString(prefix)
					content.WriteString(wline)
				}
			}
		}

		// Metadata line: due badge + created date.
		content.WriteString("\n")
		currentLine++

		var meta strings.Builder
		if note.Due != nil {
			due := *note.Due
			dueStr := due.Format("2006-01-02")
			badge := "[" + dueStr + "]"
			if due.Before(now) {
				meta.WriteString(overdueStyle.Render(badge))
			} else if due.Sub(now) <= 48*time.Hour {
				meta.WriteString(dueSoonStyle.Render(badge))
			} else {
				meta.WriteString(dueBadgeStyle.Render(badge))
			}
			meta.WriteString("  ")
		}
		meta.WriteString(createdStyle.Render(note.Created.Format("Jan 2")))
		content.WriteString(prefix + meta.String())

		currentLine++ // account for the metadata line itself
	}

	// Use viewport for scrolling.
	m.listVP.Width = m.width
	m.listVP.Height = height

	savedOffset := m.listVP.YOffset
	m.listVP.SetContent(content.String())
	m.listVP.SetYOffset(savedOffset)

	// Ensure cursor is visible using line offsets.
	if m.cursor >= 0 && m.cursor < len(noteStartLines) {
		cursorLine := noteStartLines[m.cursor]
		var cursorHeight int
		if m.cursor+1 < len(noteStartLines) {
			cursorHeight = noteStartLines[m.cursor+1] - cursorLine
		} else {
			// Last note: height = total lines - start line.
			cursorHeight = currentLine - cursorLine
		}
		if cursorLine < m.listVP.YOffset {
			m.listVP.SetYOffset(cursorLine)
		}
		if cursorLine+cursorHeight > m.listVP.YOffset+height {
			m.listVP.SetYOffset(cursorLine + cursorHeight - height)
		}
	}

	return m.listVP.View()
}

// inputLabel returns the label for the current input mode.
func (m Model) inputLabel() string {
	switch m.mode {
	case inputNewText:
		return "New note:"
	case inputNewDate:
		return "Due date:"
	case inputEditText:
		return "Edit text:"
	case inputEditDate:
		return "Due date:"
	default:
		return ""
	}
}

// sortNotes sorts notes: due-date notes first (earliest first), then no-due-date notes (newest first).
func sortNotes(notes []sdk.NoteResponse) []sdk.NoteResponse {
	sort.SliceStable(notes, func(i, j int) bool {
		iHasDue := notes[i].Due != nil
		jHasDue := notes[j].Due != nil
		if iHasDue && jHasDue {
			return notes[i].Due.Before(*notes[j].Due)
		}
		if iHasDue != jHasDue {
			return iHasDue
		}
		// Both have no due date — newest first.
		return notes[i].Created.After(notes[j].Created)
	})
	return notes
}

// wrapToWidth word-wraps text to fit within maxWidth using lipgloss.Width for
// accurate character width measurement. Returns one string per wrapped line.
func wrapToWidth(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{text}
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	var current strings.Builder
	currentWidth := 0

	for _, word := range words {
		wordWidth := lipgloss.Width(word)
		if currentWidth == 0 {
			// First word on the line — always accept it.
			current.WriteString(word)
			currentWidth = wordWidth
		} else if currentWidth+1+wordWidth <= maxWidth {
			// Fits on current line with a space.
			current.WriteString(" ")
			current.WriteString(word)
			currentWidth += 1 + wordWidth
		} else {
			// Doesn't fit — start a new line.
			lines = append(lines, current.String())
			current.Reset()
			current.WriteString(word)
			currentWidth = wordWidth
		}
	}
	lines = append(lines, current.String())
	return lines
}

// --- API commands ---

func (m Model) loadNotes() tea.Cmd {
	return func() tea.Msg {
		resp, err := m.client.ListNotes()
		if err != nil {
			return NotesErrorMsg{Err: err}
		}
		return NotesLoadedMsg{Notes: resp.Notes}
	}
}

func (m Model) createNote(text string, due *string) tea.Cmd {
	return func() tea.Msg {
		_, err := m.client.CreateNote(text, due)
		return NoteCreatedMsg{Err: err}
	}
}

func (m Model) updateNote(id string, text *string, due *string) tea.Cmd {
	return func() tea.Msg {
		_, err := m.client.UpdateNote(id, text, due)
		return NoteUpdatedMsg{Err: err}
	}
}

func (m Model) deleteNote(id string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.DeleteNote(id)
		return NoteDeletedMsg{Err: err}
	}
}

// --- SSE commands (copied from docs pattern) ---

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

func (m Model) scheduleSSEReconnect() tea.Cmd {
	return tea.Tick(m.sseBackoff, func(time.Time) tea.Msg {
		return sseReconnectTickMsg{}
	})
}

func (m Model) startPollTicker() tea.Cmd {
	return tea.Tick(pollInterval, func(time.Time) tea.Msg {
		return pollTickMsg{}
	})
}

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
