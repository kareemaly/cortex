package docs

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
	paneExplorer = 0
	panePreview  = 1
)

// categoryNode holds a category and its docs.
type categoryNode struct {
	name     string
	expanded bool
	docs     []sdk.DocSummary
}

// treeItem represents a single row in the flattened tree.
type treeItem struct {
	isCategory bool
	catIndex   int
	doc        *sdk.DocSummary
}

// Model is the Bubbletea model for the docs browser.
type Model struct {
	client     *sdk.Client
	categories []categoryNode
	tree       []treeItem
	cursor     int
	focusPane  int

	explorerVP viewport.Model
	previewVP  viewport.Model
	mdRenderer *glamour.TermRenderer

	cachedDoc   *sdk.DocResponse
	cachedDocID string

	width, height int
	ready         bool
	loading       bool
	err           error
	pendingG      bool

	// SSE subscription state.
	eventCh      <-chan sdk.Event
	cancelEvents context.CancelFunc

	// Log viewer state.
	logBuf        *tuilog.Buffer
	logViewer     tuilog.Viewer
	showLogViewer bool

	statusMsg     string
	statusIsError bool
}

// Message types for async operations.

// DocsLoadedMsg is sent when docs are successfully fetched.
type DocsLoadedMsg struct {
	Docs []sdk.DocSummary
}

// DocsErrorMsg is sent when fetching docs fails.
type DocsErrorMsg struct {
	Err error
}

// DocFetchedMsg is sent when a full doc is fetched.
type DocFetchedMsg struct {
	Doc *sdk.DocResponse
}

// DocFetchErrorMsg is sent when fetching a doc fails.
type DocFetchErrorMsg struct {
	Err error
}

// DocEditMsg is sent when a doc edit action completes.
type DocEditMsg struct {
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

// New creates a new docs model.
func New(client *sdk.Client, logBuf *tuilog.Buffer) Model {
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)
	return Model{
		client:     client,
		loading:    true,
		focusPane:  paneExplorer,
		mdRenderer: renderer,
		logBuf:     logBuf,
		logViewer:  tuilog.NewViewer(logBuf),
	}
}

// Init starts loading docs and subscribing to events.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadDocs(), m.subscribeEvents())
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
		m.updateRendererWidth()
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case DocsLoadedMsg:
		m.loading = false
		m.err = nil
		m.buildTree(msg.Docs)
		m.logBuf.Debug("api", "docs loaded")
		// Fetch the doc under cursor if any.
		return m, m.fetchSelectedDoc()

	case DocsErrorMsg:
		m.loading = false
		m.err = msg.Err
		m.logBuf.Errorf("api", "failed to load docs: %s", msg.Err)
		return m, nil

	case DocFetchedMsg:
		m.cachedDoc = msg.Doc
		m.cachedDocID = msg.Doc.ID
		m.renderPreview()
		m.logBuf.Debugf("api", "doc fetched: %s", msg.Doc.Title)
		return m, nil

	case DocFetchErrorMsg:
		m.statusMsg = fmt.Sprintf("Error loading doc: %s", msg.Err)
		m.statusIsError = true
		m.logBuf.Errorf("api", "failed to fetch doc: %s", msg.Err)
		return m, m.clearStatusAfterDelay()

	case DocEditMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Edit failed: %s", msg.Err)
			m.statusIsError = true
		} else {
			m.statusMsg = "Opened in editor"
			m.statusIsError = false
		}
		return m, m.clearStatusAfterDelay()

	case ClearStatusMsg:
		m.statusMsg = ""
		m.statusIsError = false
		return m, nil

	case sseConnectedMsg:
		m.eventCh = msg.ch
		m.cancelEvents = msg.cancel
		m.logBuf.Info("sse", "connected to event stream")
		return m, m.waitForEvent()

	case EventMsg:
		m.logBuf.Debug("sse", "event received")
		return m, tea.Batch(m.loadDocs(), m.waitForEvent())
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
			return m, m.loadDocs()
		}
		return m, nil
	}

	// Refresh.
	if isKey(msg, KeyR) {
		m.loading = true
		m.cachedDoc = nil
		m.cachedDocID = ""
		return m, m.loadDocs()
	}

	// Pane switching with h/l.
	if isKey(msg, KeyH) {
		if m.focusPane == panePreview {
			m.focusPane = paneExplorer
		}
		return m, nil
	}
	if isKey(msg, KeyL) {
		if m.focusPane == paneExplorer {
			// If cursor is on a category, expand it instead.
			if m.cursor >= 0 && m.cursor < len(m.tree) && m.tree[m.cursor].isCategory {
				return m.toggleCategory()
			}
			m.focusPane = panePreview
		}
		return m, nil
	}

	// Handle 'G' - jump to last.
	if isKey(msg, KeyShiftG) {
		m.pendingG = false
		if m.focusPane == paneExplorer {
			if len(m.tree) > 0 {
				m.cursor = len(m.tree) - 1
				return m, m.fetchSelectedDoc()
			}
		} else {
			m.previewVP.GotoBottom()
		}
		return m, nil
	}

	// Handle 'g' key for 'gg' sequence.
	if isKey(msg, KeyG) {
		if m.pendingG {
			m.pendingG = false
			if m.focusPane == paneExplorer {
				m.cursor = 0
				return m, m.fetchSelectedDoc()
			}
			m.previewVP.GotoTop()
		} else {
			m.pendingG = true
		}
		return m, nil
	}

	// Clear pending g on any other key.
	m.pendingG = false

	// Navigation.
	if m.focusPane == paneExplorer {
		return m.handleExplorerKey(msg)
	}
	return m.handlePreviewKey(msg)
}

// handleExplorerKey handles keys when the explorer pane is focused.
func (m Model) handleExplorerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case isKey(msg, KeyJ, KeyDown):
		if m.cursor < len(m.tree)-1 {
			m.cursor++
			return m, m.fetchSelectedDoc()
		}
	case isKey(msg, KeyK, KeyUp):
		if m.cursor > 0 {
			m.cursor--
			return m, m.fetchSelectedDoc()
		}
	case isKey(msg, KeyE):
		return m.editSelectedDoc()
	case isKey(msg, KeyCtrlD):
		m.cursor = min(m.cursor+10, max(len(m.tree)-1, 0))
		return m, m.fetchSelectedDoc()
	case isKey(msg, KeyCtrlU):
		m.cursor = max(m.cursor-10, 0)
		return m, m.fetchSelectedDoc()
	}
	return m, nil
}

// handlePreviewKey handles keys when the preview pane is focused.
func (m Model) handlePreviewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case isKey(msg, KeyJ, KeyDown):
		m.previewVP.ScrollDown(1)
	case isKey(msg, KeyK, KeyUp):
		m.previewVP.ScrollUp(1)
	case isKey(msg, KeyCtrlD):
		m.previewVP.HalfPageDown()
	case isKey(msg, KeyCtrlU):
		m.previewVP.HalfPageUp()
	}
	return m, nil
}

// toggleCategory toggles the expand/collapse state of the category under cursor.
func (m Model) toggleCategory() (tea.Model, tea.Cmd) {
	if m.cursor < 0 || m.cursor >= len(m.tree) {
		return m, nil
	}
	item := m.tree[m.cursor]
	if !item.isCategory {
		return m, nil
	}
	m.categories[item.catIndex].expanded = !m.categories[item.catIndex].expanded
	m.rebuildTree()
	// Clamp cursor.
	if m.cursor >= len(m.tree) {
		m.cursor = max(len(m.tree)-1, 0)
	}
	return m, nil
}

// View renders the docs browser.
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
		b.WriteString(loadingStyle.Render("Loading docs..."))
		return b.String()
	}

	// Calculate pane dimensions.
	explorerWidth := max(m.width*30/100, 20)
	previewWidth := max(m.width-explorerWidth, 20)
	// Height: status bar (1) + help bar (1) = 2 lines overhead.
	contentHeight := max(m.height-2, 5)

	// Render panes side by side.
	explorer := m.renderExplorer(explorerWidth, contentHeight)
	preview := m.renderPreviewPane(previewWidth, contentHeight)
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, explorer, preview))
	b.WriteString("\n")

	// Status bar.
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

	// Help bar.
	help := helpBarStyle.Render(helpText())
	badge := m.logBadge()
	if badge != "" {
		help = help + "  " + badge
	}
	b.WriteString(help)

	return b.String()
}

// renderExplorer renders the left pane with the tree view.
func (m Model) renderExplorer(width, height int) string {
	var b strings.Builder

	// Header.
	headerStyle := explorerHeaderStyle
	if m.focusPane == paneExplorer {
		headerStyle = activePaneHeaderStyle
	}
	header := headerStyle.Width(width - 2).Render("Explorer")
	b.WriteString(header)
	b.WriteString("\n")

	treeHeight := height - 1 // minus header

	if len(m.tree) == 0 {
		empty := lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true).
			Width(width - 2).
			Render("No docs found")
		return lipgloss.NewStyle().Width(width).Height(height).Render(b.String() + empty)
	}

	// Render tree items into a string with left-border indicator.
	var content strings.Builder
	itemWidth := max(width-5, 10) // reserve 1 char for indicator
	itemStartLines := make([]int, len(m.tree))
	currentLine := 0
	for i, item := range m.tree {
		itemStartLines[i] = currentLine
		line := m.renderTreeItem(item, itemWidth)
		var indicator string
		if i == m.cursor && m.focusPane == paneExplorer {
			indicator = selectedIndicator.Render("▎")
			line = selectedStyle.Render(line)
		} else if i == m.cursor {
			// Cursor visible but not focused — dim indicator.
			indicator = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("▎")
			line = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Render(line)
		} else {
			indicator = " "
		}
		rendered := indicator + line
		content.WriteString(rendered)
		currentLine += lipgloss.Height(rendered)
		if i < len(m.tree)-1 {
			content.WriteString("\n")
		}
	}

	// Use viewport for scrolling.
	m.explorerVP.Width = width - 2
	m.explorerVP.Height = treeHeight

	savedOffset := m.explorerVP.YOffset
	m.explorerVP.SetContent(content.String())
	m.explorerVP.SetYOffset(savedOffset)

	// Ensure cursor is visible using line offsets.
	if m.cursor >= 0 && m.cursor < len(itemStartLines) {
		cursorLine := itemStartLines[m.cursor]
		cursorHeight := 1
		if m.cursor+1 < len(itemStartLines) {
			cursorHeight = itemStartLines[m.cursor+1] - cursorLine
		}
		if cursorLine < m.explorerVP.YOffset {
			m.explorerVP.SetYOffset(cursorLine)
		}
		if cursorLine+cursorHeight > m.explorerVP.YOffset+treeHeight {
			m.explorerVP.SetYOffset(cursorLine + cursorHeight - treeHeight)
		}
	}

	b.WriteString(m.explorerVP.View())

	return lipgloss.NewStyle().Width(width).Height(height).Render(b.String())
}

// renderTreeItem renders a single tree item (category or doc).
func (m Model) renderTreeItem(item treeItem, width int) string {
	if item.isCategory {
		cat := m.categories[item.catIndex]
		arrow := "▼ "
		if !cat.expanded {
			arrow = "▶ "
		}
		name := cat.name
		count := fmt.Sprintf(" (%d)", len(cat.docs))
		color := categoryColor(item.catIndex)
		return categoryStyle.Foreground(color).Render(arrow+name) +
			lipgloss.NewStyle().Foreground(mutedColor).Render(count)
	}

	// Doc item — indented with tree connector.
	connector := treeConnector.Render("  ├─ ")
	title := item.doc.Title
	maxTitle := max(width-6, 5)
	truncated := truncateToWidth(title, maxTitle)
	return connector + docTitleStyle.Render(truncated)
}

// renderPreviewPane renders the right pane with markdown preview.
func (m Model) renderPreviewPane(width, height int) string {
	var b strings.Builder

	// Header.
	headerStyle := previewHeaderStyle
	if m.focusPane == panePreview {
		headerStyle = activePaneHeaderStyle
	}

	headerTitle := "Preview"
	if m.cachedDoc != nil {
		headerTitle = m.cachedDoc.Title
		maxLen := max(width-4, 10)
		if len(headerTitle) > maxLen {
			headerTitle = headerTitle[:maxLen-3] + "..."
		}
	}
	header := headerStyle.Width(width - 2).Render(headerTitle)
	b.WriteString(header)
	b.WriteString("\n")

	// Attribute bar height.
	attrBarHeight := 0
	var attrBar string
	if m.cachedDoc != nil {
		attrBar = m.renderAttributeBar(width - 2)
		attrBarHeight = lipgloss.Height(attrBar)
	}

	previewHeight := max(height-1-attrBarHeight, 3) // minus header and attr bar

	if m.cachedDoc == nil {
		empty := emptyPreviewStyle.Render("Select a doc to preview")
		return lipgloss.NewStyle().Width(width).Height(height).
			Render(b.String() + empty)
	}

	// Preview viewport.
	m.previewVP.Width = width - 2
	m.previewVP.Height = previewHeight
	b.WriteString(m.previewVP.View())
	b.WriteString("\n")

	// Attribute bar.
	b.WriteString(attrBar)

	return lipgloss.NewStyle().Width(width).Height(height).Render(b.String())
}

// renderAttributeBar renders the category, tags, dates, and references.
func (m Model) renderAttributeBar(width int) string {
	if m.cachedDoc == nil {
		return ""
	}
	doc := m.cachedDoc

	var parts []string

	// Category badge.
	catColor := categoryColorByName(doc.Category, m.categories)
	badge := categoryBadgeStyle.
		Foreground(lipgloss.Color("255")).
		Background(catColor).
		Render(doc.Category)
	parts = append(parts, badge)

	// Tags as pills.
	for _, tag := range doc.Tags {
		parts = append(parts, tagPillStyle.Render(tag))
	}

	// Dates.
	if doc.Created != "" {
		parts = append(parts, dateStyle.Render("created: "+formatDocDate(doc.Created)))
	}
	if doc.Updated != "" && doc.Updated != doc.Created {
		parts = append(parts, dateStyle.Render("updated: "+formatDocDate(doc.Updated)))
	}

	// References.
	for _, ref := range doc.References {
		parts = append(parts, refStyle.Render(ref))
	}

	sep := attrSeparator.Render(" | ")
	line := strings.Join(parts, sep)

	// Truncate if too wide.
	if lipgloss.Width(line) > width {
		return lipgloss.NewStyle().Width(width).Render(line)
	}
	return line
}

// buildTree groups docs by category and builds the flat tree.
func (m *Model) buildTree(docs []sdk.DocSummary) {
	// Group by category.
	catMap := make(map[string][]sdk.DocSummary)
	for _, d := range docs {
		cat := d.Category
		if cat == "" {
			cat = "uncategorized"
		}
		catMap[cat] = append(catMap[cat], d)
	}

	// Sort categories alphabetically.
	var catNames []string
	for name := range catMap {
		catNames = append(catNames, name)
	}
	sort.Strings(catNames)

	// Build category nodes. Sort docs within each by created desc.
	// Save old expanded states before resetting.
	oldExpanded := make(map[string]bool)
	for _, cat := range m.categories {
		oldExpanded[cat.name] = cat.expanded
	}

	m.categories = make([]categoryNode, 0, len(catNames))
	for _, name := range catNames {
		catDocs := catMap[name]
		sort.Slice(catDocs, func(i, j int) bool {
			return catDocs[i].Created > catDocs[j].Created
		})

		// Preserve expanded state if category already existed.
		expanded := true
		if prev, ok := oldExpanded[name]; ok {
			expanded = prev
		}

		m.categories = append(m.categories, categoryNode{
			name:     name,
			expanded: expanded,
			docs:     catDocs,
		})
	}

	m.rebuildTree()
}

// rebuildTree flattens categories + docs into the tree slice.
func (m *Model) rebuildTree() {
	m.tree = nil
	for i, cat := range m.categories {
		m.tree = append(m.tree, treeItem{
			isCategory: true,
			catIndex:   i,
		})
		if cat.expanded {
			for j := range cat.docs {
				m.tree = append(m.tree, treeItem{
					catIndex: i,
					doc:      &m.categories[i].docs[j],
				})
			}
		}
	}
}

// selectedDoc returns the DocSummary at the current cursor, or nil if on a category.
func (m *Model) selectedDoc() *sdk.DocSummary {
	if m.cursor < 0 || m.cursor >= len(m.tree) {
		return nil
	}
	item := m.tree[m.cursor]
	if item.isCategory {
		return nil
	}
	return item.doc
}

// editSelectedDoc opens the selected doc in $EDITOR via tmux popup.
func (m Model) editSelectedDoc() (tea.Model, tea.Cmd) {
	doc := m.selectedDoc()
	if doc == nil {
		return m, nil
	}
	id := doc.ID
	client := m.client
	return m, func() tea.Msg {
		err := client.OpenDocInEditor(id)
		return DocEditMsg{Err: err}
	}
}

// fetchSelectedDoc returns a command to fetch the full doc if the cursor is on a doc.
func (m *Model) fetchSelectedDoc() tea.Cmd {
	doc := m.selectedDoc()
	if doc == nil {
		// Clear preview when on category.
		m.cachedDoc = nil
		m.cachedDocID = ""
		m.previewVP.SetContent("")
		return nil
	}
	// Skip if already cached.
	if doc.ID == m.cachedDocID {
		return nil
	}
	return m.fetchDoc(doc.ID)
}

// renderPreview renders the cached doc body into the preview viewport.
func (m *Model) renderPreview() {
	if m.cachedDoc == nil {
		m.previewVP.SetContent("")
		return
	}

	body := m.cachedDoc.Body
	if body == "" {
		m.previewVP.SetContent(emptyPreviewStyle.Render("(empty)"))
		return
	}

	if m.mdRenderer != nil {
		rendered, err := m.mdRenderer.Render(body)
		if err == nil {
			m.previewVP.SetContent(strings.TrimSpace(rendered))
			return
		}
	}
	m.previewVP.SetContent(body)
}

// updateRendererWidth recreates the glamour renderer to match the preview width.
func (m *Model) updateRendererWidth() {
	previewWidth := max(m.width*70/100-4, 40)
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(previewWidth),
	)
	m.mdRenderer = renderer
	// Re-render if we have cached content.
	if m.cachedDoc != nil {
		m.renderPreview()
	}
}

// loadDocs returns a command to load all docs.
func (m Model) loadDocs() tea.Cmd {
	return func() tea.Msg {
		resp, err := m.client.ListDocs("", "", "")
		if err != nil {
			return DocsErrorMsg{Err: err}
		}
		return DocsLoadedMsg{Docs: resp.Docs}
	}
}

// fetchDoc returns a command to fetch a full doc by ID.
func (m Model) fetchDoc(id string) tea.Cmd {
	return func() tea.Msg {
		doc, err := m.client.GetDoc(id)
		if err != nil {
			return DocFetchErrorMsg{Err: err}
		}
		return DocFetchedMsg{Doc: doc}
	}
}

// subscribeEvents returns a command that connects to the SSE event stream.
func (m Model) subscribeEvents() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		ch, err := m.client.SubscribeEvents(ctx)
		if err != nil {
			cancel()
			return nil
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

// formatDocDate formats a date string for display.
func formatDocDate(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		// Try other common formats.
		t, err = time.Parse(time.RFC3339Nano, s)
		if err != nil {
			return s
		}
	}
	return t.Format("Jan 2, 2006")
}

// categoryColorByName returns the color for a named category.
func categoryColorByName(name string, cats []categoryNode) lipgloss.Color {
	for i, cat := range cats {
		if cat.name == name {
			return categoryColor(i)
		}
	}
	return categoryColors[0]
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
