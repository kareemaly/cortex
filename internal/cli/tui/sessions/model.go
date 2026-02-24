package sessions

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/cli/tui/tuilog"
)

const (
	sseInitialBackoff = 2 * time.Second
	sseMaxBackoff     = 30 * time.Second
	pollInterval      = 60 * time.Second
)

type viewMode int

const (
	viewList viewMode = iota
	viewDetail
)

// dateGroup holds sessions for a single calendar date.
type dateGroup struct {
	date     time.Time
	sessions []sdk.ConclusionSummary
}

type Model struct {
	client *sdk.Client

	// Date-grouped state (replaces flat conclusions slice)
	dateGroups []dateGroup
	dateIdx    int // index into dateGroups
	cursor     int // row within dateGroups[dateIdx].sessions

	detailVP viewport.Model
	mode     viewMode

	width, height int
	ready         bool
	loading       bool
	err           error
	pendingG      bool

	mdRenderer *glamour.TermRenderer

	eventCh      <-chan sdk.Event
	cancelEvents context.CancelFunc
	sseBackoff   time.Duration
	sseConnected bool

	logBuf        *tuilog.Buffer
	logViewer     tuilog.Viewer
	showLogViewer bool

	statusMsg     string
	statusIsError bool
}

// Message types

type ConclusionsLoadedMsg struct {
	Conclusions []sdk.ConclusionSummary
}

type ConclusionsErrorMsg struct {
	Err error
}

type ClearStatusMsg struct{}

type sseConnectedMsg struct {
	ch     <-chan sdk.Event
	cancel context.CancelFunc
}

type EventMsg struct{}

type sseDisconnectedMsg struct{}

type sseReconnectTickMsg struct{}

type pollTickMsg struct{}

type openEditorMsg struct{}

type openEditorErrMsg struct{ Err error }

func New(client *sdk.Client, logBuf *tuilog.Buffer) Model {
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)
	return Model{
		client:     client,
		loading:    true,
		mode:       viewList,
		mdRenderer: renderer,
		logBuf:     logBuf,
		logViewer:  tuilog.NewViewer(logBuf),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadConclusions(), m.subscribeEvents(), m.startPollTicker())
}

func (m Model) InputActive() bool {
	return false
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tuilog.DismissLogViewerMsg); ok {
		m.showLogViewer = false
		return m, nil
	}

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

	case ConclusionsLoadedMsg:
		m.loading = false
		m.err = nil
		m.dateGroups = groupByDate(msg.Conclusions)
		// Clamp date index
		if m.dateIdx >= len(m.dateGroups) {
			m.dateIdx = max(len(m.dateGroups)-1, 0)
		}
		// Clamp cursor
		if len(m.dateGroups) > 0 {
			if m.cursor >= len(m.dateGroups[m.dateIdx].sessions) {
				m.cursor = max(len(m.dateGroups[m.dateIdx].sessions)-1, 0)
			}
		} else {
			m.cursor = 0
		}
		m.logBuf.Debug("api", "conclusions loaded")
		return m, nil

	case ConclusionsErrorMsg:
		m.loading = false
		m.err = msg.Err
		m.logBuf.Errorf("api", "failed to load conclusions: %s", msg.Err)
		return m, nil

	case ClearStatusMsg:
		m.statusMsg = ""
		m.statusIsError = false
		return m, nil

	case openEditorMsg:
		m.statusMsg = "Editor closed"
		m.statusIsError = false
		return m, clearStatusAfter(3 * time.Second)

	case openEditorErrMsg:
		m.statusMsg = fmt.Sprintf("Error: %s", msg.Err)
		m.statusIsError = true
		return m, clearStatusAfter(5 * time.Second)

	case sseConnectedMsg:
		if m.cancelEvents != nil {
			m.cancelEvents()
		}
		m.eventCh = msg.ch
		m.cancelEvents = msg.cancel
		m.sseConnected = true
		m.sseBackoff = 0
		m.logBuf.Info("sse", "connected to event stream")
		return m, tea.Batch(m.loadConclusions(), m.waitForEvent())

	case EventMsg:
		m.logBuf.Debug("sse", "event received")
		return m, tea.Batch(m.loadConclusions(), m.waitForEvent())

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
		return m, tea.Batch(m.loadConclusions(), m.startPollTicker())
	}

	return m, nil
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if isKey(msg, KeyCtrlC) {
		if m.cancelEvents != nil {
			m.cancelEvents()
		}
		return m, tea.Quit
	}

	if isKey(msg, KeyBang) {
		m.showLogViewer = !m.showLogViewer
		if m.showLogViewer {
			m.logViewer.SetSize(m.width, m.height)
			m.logViewer.Reset()
		}
		return m, nil
	}

	if m.loading {
		return m, nil
	}

	if m.err != nil {
		if isKey(msg, KeyR) {
			m.loading = true
			m.err = nil
			return m, m.loadConclusions()
		}
		if isKey(msg, KeyQuit) {
			if m.cancelEvents != nil {
				m.cancelEvents()
			}
			return m, tea.Quit
		}
		return m, nil
	}

	if m.mode == viewDetail {
		return m.handleDetailKey(msg)
	}

	return m.handleListKey(msg)
}

func (m Model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if isKey(msg, KeyQuit) {
		if m.cancelEvents != nil {
			m.cancelEvents()
		}
		return m, tea.Quit
	}

	if isKey(msg, KeyR) {
		m.loading = true
		return m, m.loadConclusions()
	}

	// Date navigation
	if isKey(msg, KeyLeft) {
		if m.dateIdx > 0 {
			m.dateIdx--
			m.cursor = 0
		}
		return m, nil
	}
	if isKey(msg, KeyRight) {
		if m.dateIdx < len(m.dateGroups)-1 {
			m.dateIdx++
			m.cursor = 0
		}
		return m, nil
	}

	if isKey(msg, KeyShiftG) {
		m.pendingG = false
		if len(m.dateGroups) > 0 {
			m.cursor = max(len(m.dateGroups[m.dateIdx].sessions)-1, 0)
		}
		return m, nil
	}

	if isKey(msg, KeyG) {
		if m.pendingG {
			m.pendingG = false
			m.cursor = 0
		} else {
			m.pendingG = true
		}
		return m, nil
	}

	m.pendingG = false

	// Open editor
	if isKey(msg, KeyOpenEditor, KeyEnter) {
		if len(m.dateGroups) > 0 && m.cursor >= 0 && m.cursor < len(m.dateGroups[m.dateIdx].sessions) {
			c := m.dateGroups[m.dateIdx].sessions[m.cursor]
			return m, m.openConclusionInEditor(c)
		}
		return m, nil
	}

	switch {
	case isKey(msg, KeyJ, KeyDown):
		if len(m.dateGroups) > 0 && m.cursor < len(m.dateGroups[m.dateIdx].sessions)-1 {
			m.cursor++
		}
	case isKey(msg, KeyK, KeyUp):
		if m.cursor > 0 {
			m.cursor--
		}
	case isKey(msg, KeyCtrlD):
		if len(m.dateGroups) > 0 {
			m.cursor = min(m.cursor+10, max(len(m.dateGroups[m.dateIdx].sessions)-1, 0))
		}
	case isKey(msg, KeyCtrlU):
		m.cursor = max(m.cursor-10, 0)
	}

	return m, nil
}

func (m Model) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if isKey(msg, KeyEsc, KeyQuit) {
		m.mode = viewList
		return m, nil
	}

	if isKey(msg, KeyShiftG) {
		m.pendingG = false
		m.detailVP.GotoBottom()
		return m, nil
	}

	if isKey(msg, KeyG) {
		if m.pendingG {
			m.pendingG = false
			m.detailVP.GotoTop()
		} else {
			m.pendingG = true
		}
		return m, nil
	}

	m.pendingG = false

	switch {
	case isKey(msg, KeyJ, KeyDown):
		m.detailVP.ScrollDown(1)
	case isKey(msg, KeyK, KeyUp):
		m.detailVP.ScrollUp(1)
	case isKey(msg, KeyCtrlD):
		m.detailVP.HalfPageDown()
	case isKey(msg, KeyCtrlU):
		m.detailVP.HalfPageUp()
	}

	return m, nil
}

func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	if m.showLogViewer {
		return m.logViewer.View()
	}

	var b strings.Builder

	if m.err != nil {
		b.WriteString(errorStatusStyle.Render(fmt.Sprintf("Error: %s", m.err)))
		b.WriteString("\n\n")
		b.WriteString("Press [r] to retry or [q] to quit\n")
		if strings.Contains(m.err.Error(), "connect") {
			b.WriteString("\nIs the daemon running? Start it with: cortexd start\n")
		}
		return b.String()
	}

	if m.loading {
		b.WriteString(loadingStyle.Render("Loading sessions..."))
		return b.String()
	}

	contentHeight := max(m.height-3, 3)

	if m.mode == viewDetail {
		b.WriteString(m.renderDetailView(contentHeight))
	} else {
		b.WriteString(m.renderListView(contentHeight))
	}

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

	// Status bar
	var sessionCount int
	if len(m.dateGroups) > 0 && m.dateIdx < len(m.dateGroups) {
		sessionCount = len(m.dateGroups[m.dateIdx].sessions)
	}
	countStr := statusBarStyle.Render(fmt.Sprintf("%d sessions", sessionCount))

	var helpStr string
	if m.mode == viewDetail {
		helpStr = countStr + "  " + helpBarStyle.Render(detailHelpText())
	} else {
		openHint := helpBarStyle.Render("o/↵: open")
		helpStr = countStr + "  " + helpBarStyle.Render(listHelpText()) + "  " + openHint
	}

	badge := m.logBadge()
	if badge != "" {
		helpStr = helpStr + "  " + badge
	}
	b.WriteString(helpStr)

	return b.String()
}

// renderListView composes date strip + divider + session list.
func (m Model) renderListView(height int) string {
	if len(m.dateGroups) == 0 {
		empty := emptyStyle.Padding(1, 2).Render("No concluded sessions yet.")
		return lipgloss.NewStyle().Width(m.width).Height(height).Render(empty)
	}

	dateStrip := m.renderDateStrip()
	stripHeight := strings.Count(dateStrip, "\n") + 1

	divider := dividerStyle.Render(strings.Repeat("─", m.width))
	dividerHeight := 1

	listHeight := max(height-stripHeight-dividerHeight-1, 1)

	sessionList := m.renderSessionList(listHeight)

	return dateStripStyle.Render(dateStrip) + "\n" + divider + "\n" + sessionList
}

// renderDateStrip renders the horizontal date strip with active/inactive pills.
func (m Model) renderDateStrip() string {
	if len(m.dateGroups) == 0 {
		return ""
	}

	var parts []string
	for i, dg := range m.dateGroups {
		label := dg.date.Format("Jan 2")
		if i == m.dateIdx {
			parts = append(parts, activeDateStyle.Render(label))
		} else {
			parts = append(parts, inactiveDateStyle.Render(label))
		}
	}

	strip := strings.Join(parts, " ")

	// Add navigation arrows
	leftArrow := "◀ "
	rightArrow := " ▶"
	if m.dateIdx == 0 {
		leftArrow = "  "
	}
	if m.dateIdx == len(m.dateGroups)-1 {
		rightArrow = "  "
	}

	return leftArrow + strip + rightArrow
}

// renderSessionList renders the session rows for the currently selected date.
func (m Model) renderSessionList(height int) string {
	if m.dateIdx >= len(m.dateGroups) {
		return emptyStyle.Render("No sessions for this date.")
	}

	sessions := m.dateGroups[m.dateIdx].sessions
	if len(sessions) == 0 {
		return emptyStyle.Render("No sessions for this date.")
	}

	var lines []string
	for i, c := range sessions {
		timeStr := timeStyle.Render(c.Created.Local().Format("15:04"))

		typeLabel := typeShortLabel(c.Type)
		typePart := typeLabelStyle.Render(typeLabel)

		title := sessionTitle(c)

		row := timeStr + "  " + typePart + "  " + title

		if i == m.cursor {
			row = selectedItemStyle.Render("▸ ") + selectedItemStyle.Render(row)
		} else {
			row = "  " + row
		}
		lines = append(lines, row)
	}

	// Clamp to visible height with scroll
	start := 0
	if m.cursor >= height {
		start = m.cursor - height + 1
	}
	end := min(start+height, len(lines))

	return strings.Join(lines[start:end], "\n")
}

func (m Model) renderDetailView(height int) string {
	if len(m.dateGroups) == 0 || m.dateIdx >= len(m.dateGroups) {
		return emptyStyle.Render("No session selected")
	}
	sessions := m.dateGroups[m.dateIdx].sessions
	if m.cursor < 0 || m.cursor >= len(sessions) {
		return emptyStyle.Render("No session selected")
	}

	c := sessions[m.cursor]

	var header strings.Builder
	header.WriteString(detailHeaderStyle.Render("Session Conclusion"))
	header.WriteString("\n")

	var meta strings.Builder
	meta.WriteString(typeBadgeStyle(c.Type).Render(c.Type))
	meta.WriteString(" ")
	if c.TicketTitle != "" {
		meta.WriteString(ticketRefStyle.Render(c.TicketTitle))
		meta.WriteString("  ")
	} else if c.Ticket != "" {
		meta.WriteString("Ticket: ")
		meta.WriteString(ticketRefStyle.Render(c.Ticket))
		meta.WriteString("  ")
	}
	meta.WriteString(dateStyle.Render(c.Created.Local().Format("Jan 2, 2006 15:04")))
	header.WriteString(detailMetaStyle.Render(meta.String()))
	header.WriteString("\n\n")

	headerStr := header.String()
	headerLines := strings.Count(headerStr, "\n")
	vpHeight := max(height-headerLines, 3)

	m.detailVP.Width = m.width
	m.detailVP.Height = vpHeight

	body := "(body not available in list view — use the readConclusion MCP tool to view full content)"
	rendered := m.renderMarkdown(body)
	m.detailVP.SetContent(rendered)

	return headerStr + m.detailVP.View()
}

func (m Model) renderMarkdown(content string) string {
	if m.mdRenderer == nil {
		return content
	}
	rendered, err := m.mdRenderer.Render(content)
	if err != nil {
		return content
	}
	return strings.TrimSpace(rendered)
}

// openConclusionInEditor returns a Cmd that opens the conclusion's index.md in $EDITOR.
func (m Model) openConclusionInEditor(c sdk.ConclusionSummary) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.EditConclusion(c.ID); err != nil {
			return openEditorErrMsg{Err: fmt.Errorf("open editor: %w", err)}
		}
		return openEditorMsg{}
	}
}

// groupByDate groups conclusions by local calendar date, newest date first.
func groupByDate(conclusions []sdk.ConclusionSummary) []dateGroup {
	byDate := make(map[string]*dateGroup)
	var order []string

	for _, c := range conclusions {
		local := c.Created.Local()
		key := local.Format("2006-01-02")
		if _, exists := byDate[key]; !exists {
			dayStart := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location())
			byDate[key] = &dateGroup{date: dayStart}
			order = append(order, key)
		}
		byDate[key].sessions = append(byDate[key].sessions, c)
	}

	// Sort each group newest-first within the day
	for _, dg := range byDate {
		sort.SliceStable(dg.sessions, func(i, j int) bool {
			return dg.sessions[i].Created.After(dg.sessions[j].Created)
		})
	}

	// Sort date keys newest date first
	sort.Slice(order, func(i, j int) bool {
		return order[i] > order[j]
	})

	groups := make([]dateGroup, 0, len(order))
	for _, key := range order {
		groups = append(groups, *byDate[key])
	}
	return groups
}

// typeShortLabel returns a short human-readable label for a conclusion type.
func typeShortLabel(t string) string {
	switch t {
	case "architect":
		return "arch"
	case "research":
		return "research"
	case "work":
		return "work"
	default:
		return t
	}
}

// sessionTitle returns the display title for a conclusion.
func sessionTitle(c sdk.ConclusionSummary) string {
	if c.TicketTitle != "" {
		return c.TicketTitle
	}
	if c.Type == "architect" {
		return "Architect session"
	}
	if c.Ticket != "" {
		return c.Ticket
	}
	return c.ID
}

func (m Model) loadConclusions() tea.Cmd {
	return func() tea.Msg {
		resp, err := m.client.ListConclusions(sdk.ListConclusionsParams{})
		if err != nil {
			return ConclusionsErrorMsg{Err: err}
		}
		return ConclusionsLoadedMsg{Conclusions: resp.Conclusions}
	}
}

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

func clearStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return ClearStatusMsg{}
	})
}
