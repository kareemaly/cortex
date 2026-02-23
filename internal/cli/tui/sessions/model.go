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

type Model struct {
	client      *sdk.Client
	conclusions []sdk.ConclusionResponse
	cursor      int

	listVP   viewport.Model
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

type ConclusionsLoadedMsg struct {
	Conclusions []sdk.ConclusionResponse
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
		m.conclusions = sortConclusions(msg.Conclusions)
		if m.cursor >= len(m.conclusions) {
			m.cursor = max(len(m.conclusions)-1, 0)
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

	if isKey(msg, KeyShiftG) {
		m.pendingG = false
		if len(m.conclusions) > 0 {
			m.cursor = len(m.conclusions) - 1
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

	switch {
	case isKey(msg, KeyJ, KeyDown):
		if m.cursor < len(m.conclusions)-1 {
			m.cursor++
		}
	case isKey(msg, KeyK, KeyUp):
		if m.cursor > 0 {
			m.cursor--
		}
	case isKey(msg, KeyCtrlD):
		m.cursor = min(m.cursor+10, max(len(m.conclusions)-1, 0))
	case isKey(msg, KeyCtrlU):
		m.cursor = max(m.cursor-10, 0)

	case isKey(msg, KeyEnter):
		if m.cursor >= 0 && m.cursor < len(m.conclusions) {
			m.mode = viewDetail
			m.updateDetailViewport()
		}
		return m, nil
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

	count := statusBarStyle.Render(fmt.Sprintf("%d sessions", len(m.conclusions)))
	var help string
	if m.mode == viewDetail {
		help = count + "  " + helpBarStyle.Render(detailHelpText())
	} else {
		help = count + "  " + helpBarStyle.Render(listHelpText())
	}
	badge := m.logBadge()
	if badge != "" {
		help = help + "  " + badge
	}
	b.WriteString(help)

	return b.String()
}

func (m Model) renderListView(height int) string {
	if len(m.conclusions) == 0 {
		empty := emptyStyle.Padding(1, 2).Render("No concluded sessions yet.")
		return lipgloss.NewStyle().Width(m.width).Height(height).Render(empty)
	}

	var content strings.Builder
	prefix := "  "
	cursorPrefix := "▸ "
	contentWidth := max(m.width-lipgloss.Width(prefix), 10)

	conclusionStartLines := make([]int, len(m.conclusions))
	currentLine := 0

	for i, c := range m.conclusions {
		conclusionStartLines[i] = currentLine

		if i > 0 {
			content.WriteString("\n")
			currentLine++
		}

		title := c.Body
		if len(title) > 60 {
			title = title[:57] + "..."
		}
		if title == "" {
			title = "(empty)"
		}

		wrapped := wrapToWidth(title, contentWidth)

		for j, wline := range wrapped {
			if j == 0 {
				if i == m.cursor {
					content.WriteString(cursorPrefix)
					content.WriteString(selectedItemStyle.Render(wline))
				} else {
					content.WriteString(prefix)
					content.WriteString(wline)
				}
			} else {
				content.WriteString("\n")
				currentLine++
				if i == m.cursor {
					content.WriteString(prefix)
					content.WriteString(selectedItemStyle.Render(wline))
				} else {
					content.WriteString(prefix)
					content.WriteString(wline)
				}
			}
		}

		content.WriteString("\n")
		currentLine++

		var meta strings.Builder
		badge := typeBadgeStyle(c.Type).Render(c.Type)
		meta.WriteString(badge)
		meta.WriteString(" ")

		if c.Ticket != "" {
			meta.WriteString(ticketRefStyle.Render(c.Ticket))
			meta.WriteString(" ")
		}
		meta.WriteString(dateStyle.Render(c.Created.Format("Jan 2, 15:04")))
		content.WriteString(prefix + meta.String())

		currentLine++
	}

	m.listVP.Width = m.width
	m.listVP.Height = height

	savedOffset := m.listVP.YOffset
	m.listVP.SetContent(content.String())
	m.listVP.SetYOffset(savedOffset)

	if m.cursor >= 0 && m.cursor < len(conclusionStartLines) {
		cursorLine := conclusionStartLines[m.cursor]
		var cursorHeight int
		if m.cursor+1 < len(conclusionStartLines) {
			cursorHeight = conclusionStartLines[m.cursor+1] - cursorLine
		} else {
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

func (m Model) renderDetailView(height int) string {
	if m.cursor < 0 || m.cursor >= len(m.conclusions) {
		return emptyStyle.Render("No session selected")
	}

	c := m.conclusions[m.cursor]

	var header strings.Builder
	header.WriteString(detailHeaderStyle.Render("Session Conclusion"))
	header.WriteString("\n")

	var meta strings.Builder
	meta.WriteString(typeBadgeStyle(c.Type).Render(c.Type))
	meta.WriteString(" ")
	if c.Ticket != "" {
		meta.WriteString("Ticket: ")
		meta.WriteString(ticketRefStyle.Render(c.Ticket))
		meta.WriteString("  ")
	}
	meta.WriteString(dateStyle.Render(c.Created.Format("Jan 2, 2006 15:04")))
	header.WriteString(detailMetaStyle.Render(meta.String()))
	header.WriteString("\n\n")

	headerStr := header.String()
	headerLines := strings.Count(headerStr, "\n")
	vpHeight := max(height-headerLines, 3)

	m.detailVP.Width = m.width
	m.detailVP.Height = vpHeight

	body := c.Body
	if body == "" {
		body = "(no content)"
	}

	rendered := m.renderMarkdown(body)
	m.detailVP.SetContent(rendered)

	return headerStr + m.detailVP.View()
}

func (m *Model) updateDetailViewport() {
	if m.cursor < 0 || m.cursor >= len(m.conclusions) {
		return
	}

	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(m.width-2),
	)
	m.mdRenderer = renderer

	vpWidth := m.width
	vpHeight := max(m.height-8, 3)
	m.detailVP = viewport.New(vpWidth, vpHeight)
	m.detailVP.SetContent("")
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

func sortConclusions(conclusions []sdk.ConclusionResponse) []sdk.ConclusionResponse {
	sorted := make([]sdk.ConclusionResponse, len(conclusions))
	copy(sorted, conclusions)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Created.After(sorted[j].Created)
	})
	return sorted
}

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
			current.WriteString(word)
			currentWidth = wordWidth
		} else if currentWidth+1+wordWidth <= maxWidth {
			current.WriteString(" ")
			current.WriteString(word)
			currentWidth += 1 + wordWidth
		} else {
			lines = append(lines, current.String())
			current.Reset()
			current.WriteString(word)
			currentWidth = wordWidth
		}
	}
	lines = append(lines, current.String())
	return lines
}

func (m Model) loadConclusions() tea.Cmd {
	return func() tea.Msg {
		resp, err := m.client.ListConclusions()
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
