package config

import (
	"fmt"
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

// listItem represents a single row in the flattened list.
type listItem struct {
	isSectionHeader bool
	isConfigFile    bool
	promptFile      *sdk.PromptFileInfo
	sectionName     string
}

// Model is the Bubbletea model for the config browser.
type Model struct {
	client *sdk.Client
	items  []listItem
	cursor int

	focusPane  int
	explorerVP viewport.Model
	previewVP  viewport.Model
	mdRenderer *glamour.TermRenderer

	promptData *sdk.ListPromptsResponse

	width, height int
	ready         bool
	loading       bool
	err           error
	pendingG      bool

	logBuf        *tuilog.Buffer
	logViewer     tuilog.Viewer
	showLogViewer bool

	statusMsg     string
	statusIsError bool

	showResetModal  bool
	resetPromptPath string
}

// Message types for async operations.

// PromptsLoadedMsg is sent when prompts are successfully fetched.
type PromptsLoadedMsg struct {
	Data *sdk.ListPromptsResponse
}

// PromptsErrorMsg is sent when fetching prompts fails.
type PromptsErrorMsg struct {
	Err error
}

// PromptEjectMsg is sent when a prompt eject action completes.
type PromptEjectMsg struct {
	Err error
}

// PromptResetMsg is sent when a prompt reset action completes.
type PromptResetMsg struct {
	Err error
}

// PromptEditMsg is sent when a prompt edit action completes.
type PromptEditMsg struct {
	Err error
}

// ConfigEditMsg is sent when a config edit action completes.
type ConfigEditMsg struct {
	Err error
}

// ClearStatusMsg clears the status message after a delay.
type ClearStatusMsg struct{}

// New creates a new config model.
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

// Init starts loading prompt data.
func (m Model) Init() tea.Cmd {
	return m.loadPrompts()
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

	case PromptsLoadedMsg:
		m.loading = false
		m.err = nil
		m.promptData = msg.Data
		m.buildItems()
		m.logBuf.Debug("api", "prompts loaded")
		m.updatePreview()
		return m, nil

	case PromptsErrorMsg:
		m.loading = false
		m.err = msg.Err
		m.logBuf.Errorf("api", "failed to load prompts: %s", msg.Err)
		return m, nil

	case PromptEjectMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Eject failed: %s", msg.Err)
			m.statusIsError = true
		} else {
			m.statusMsg = "Prompt ejected"
			m.statusIsError = false
			// Reload data
			return m, tea.Batch(m.loadPrompts(), m.clearStatusAfterDelay())
		}
		return m, m.clearStatusAfterDelay()

	case PromptResetMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Reset failed: %s", msg.Err)
			m.statusIsError = true
		} else {
			m.statusMsg = "Prompt reset to default"
			m.statusIsError = false
			return m, tea.Batch(m.loadPrompts(), m.clearStatusAfterDelay())
		}
		return m, m.clearStatusAfterDelay()

	case PromptEditMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Edit failed: %s", msg.Err)
			m.statusIsError = true
		} else {
			m.statusMsg = "Opened in editor"
			m.statusIsError = false
		}
		return m, m.clearStatusAfterDelay()

	case ConfigEditMsg:
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
	}

	return m, nil
}

// handleKeyMsg handles keyboard input.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Quit.
	if isKey(msg, KeyQuit, KeyCtrlC) {
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

	// Modal state takes priority.
	if m.showResetModal {
		return m.handleResetModalKey(msg)
	}

	// Don't process other keys while loading or if there's an error (except refresh).
	if m.loading {
		return m, nil
	}
	if m.err != nil {
		if isKey(msg, KeyR) {
			m.loading = true
			m.err = nil
			return m, m.loadPrompts()
		}
		return m, nil
	}

	// Refresh.
	if isKey(msg, KeyR) {
		m.loading = true
		return m, m.loadPrompts()
	}

	// Config shortcut.
	if isKey(msg, KeyC) {
		return m, m.editConfig()
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
			m.focusPane = panePreview
		}
		return m, nil
	}

	// Handle 'G' - jump to last.
	if isKey(msg, KeyShiftG) {
		m.pendingG = false
		if m.focusPane == paneExplorer {
			m.jumpToLast()
			m.updatePreview()
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
				m.jumpToFirst()
				m.updatePreview()
			} else {
				m.previewVP.GotoTop()
			}
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
		m.moveNext()
		m.updatePreview()
	case isKey(msg, KeyK, KeyUp):
		m.movePrev()
		m.updatePreview()
	case isKey(msg, KeyE):
		return m.handleAction()
	case isKey(msg, KeyX):
		return m.handleResetAction()
	case isKey(msg, KeyCtrlD):
		for i := 0; i < 10; i++ {
			m.moveNext()
		}
		m.updatePreview()
	case isKey(msg, KeyCtrlU):
		for i := 0; i < 10; i++ {
			m.movePrev()
		}
		m.updatePreview()
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

// buildItems builds the flat list of items from prompt data.
func (m *Model) buildItems() {
	m.items = nil
	if m.promptData == nil {
		return
	}

	// Config file first.
	m.items = append(m.items, listItem{isConfigFile: true})

	// Groups with section headers.
	for _, group := range m.promptData.Groups {
		m.items = append(m.items, listItem{
			isSectionHeader: true,
			sectionName:     group.Name,
		})
		for i := range group.Files {
			m.items = append(m.items, listItem{
				promptFile: &m.promptData.Groups[m.groupIndex(group.Key)].Files[i],
			})
		}
	}

	// Ensure cursor is on a navigable item.
	if m.cursor >= len(m.items) {
		m.cursor = 0
	}
	m.snapToNavigable()
}

// groupIndex returns the index of a group by key.
func (m *Model) groupIndex(key string) int {
	for i, g := range m.promptData.Groups {
		if g.Key == key {
			return i
		}
	}
	return 0
}

// isNavigable returns true if the item at index is navigable (not a section header).
func (m *Model) isNavigable(idx int) bool {
	if idx < 0 || idx >= len(m.items) {
		return false
	}
	return !m.items[idx].isSectionHeader
}

// snapToNavigable moves the cursor to the nearest navigable item.
func (m *Model) snapToNavigable() {
	if len(m.items) == 0 {
		m.cursor = 0
		return
	}
	if m.isNavigable(m.cursor) {
		return
	}
	// Try forward.
	for i := m.cursor + 1; i < len(m.items); i++ {
		if m.isNavigable(i) {
			m.cursor = i
			return
		}
	}
	// Try backward.
	for i := m.cursor - 1; i >= 0; i-- {
		if m.isNavigable(i) {
			m.cursor = i
			return
		}
	}
}

// moveNext moves the cursor to the next navigable item.
func (m *Model) moveNext() {
	for i := m.cursor + 1; i < len(m.items); i++ {
		if m.isNavigable(i) {
			m.cursor = i
			return
		}
	}
}

// movePrev moves the cursor to the previous navigable item.
func (m *Model) movePrev() {
	for i := m.cursor - 1; i >= 0; i-- {
		if m.isNavigable(i) {
			m.cursor = i
			return
		}
	}
}

// jumpToFirst jumps to the first navigable item.
func (m *Model) jumpToFirst() {
	for i := 0; i < len(m.items); i++ {
		if m.isNavigable(i) {
			m.cursor = i
			return
		}
	}
}

// jumpToLast jumps to the last navigable item.
func (m *Model) jumpToLast() {
	for i := len(m.items) - 1; i >= 0; i-- {
		if m.isNavigable(i) {
			m.cursor = i
			return
		}
	}
}

// currentItem returns the item under the cursor.
func (m *Model) currentItem() *listItem {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return nil
	}
	return &m.items[m.cursor]
}

// handleAction handles the 'e' key action based on the current item.
func (m Model) handleAction() (tea.Model, tea.Cmd) {
	item := m.currentItem()
	if item == nil {
		return m, nil
	}

	if item.isConfigFile {
		return m, m.editConfig()
	}

	if item.promptFile != nil {
		if item.promptFile.Ejected {
			return m, m.editPrompt(item.promptFile.Path)
		}
		return m, m.ejectPrompt(item.promptFile.Path)
	}

	return m, nil
}

// handleResetAction handles the 'x' key to initiate prompt reset.
func (m Model) handleResetAction() (tea.Model, tea.Cmd) {
	item := m.currentItem()
	if item == nil || item.promptFile == nil {
		return m, nil
	}

	if !item.promptFile.Ejected {
		m.statusMsg = "Already using default"
		m.statusIsError = false
		return m, m.clearStatusAfterDelay()
	}

	m.showResetModal = true
	m.resetPromptPath = item.promptFile.Path
	return m, nil
}

// handleResetModalKey handles keyboard input when the reset confirmation modal is shown.
func (m Model) handleResetModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case isKey(msg, KeyY):
		m.showResetModal = false
		m.statusMsg = "Resetting..."
		m.statusIsError = false
		return m, m.resetPrompt(m.resetPromptPath)

	case isKey(msg, KeyN, KeyEscape):
		m.showResetModal = false
		m.resetPromptPath = ""
		return m, nil
	}

	return m, nil
}

// resetPrompt returns a command to reset an ejected prompt to default.
func (m Model) resetPrompt(path string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		err := client.ResetPrompt(path)
		return PromptResetMsg{Err: err}
	}
}

// renderResetModal renders the reset confirmation modal prompt.
func (m Model) renderResetModal() string {
	path := m.resetPromptPath
	if len(path) > 40 {
		path = path[:37] + "..."
	}
	prompt := fmt.Sprintf("Reset \"%s\" to default?", path)
	options := "[y]es  [n]o"
	return statusBarStyle.Render(prompt) + "\n" + helpBarStyle.Render(options)
}

// updatePreview renders the preview for the current item.
func (m *Model) updatePreview() {
	item := m.currentItem()
	if item == nil {
		m.previewVP.SetContent(emptyPreviewStyle.Render("No item selected"))
		return
	}

	if item.isConfigFile && m.promptData != nil {
		body := "```yaml\n" + m.promptData.ConfigContent + "\n```"
		if m.mdRenderer != nil {
			rendered, err := m.mdRenderer.Render(body)
			if err == nil {
				m.previewVP.SetContent(strings.TrimSpace(rendered))
				m.previewVP.GotoTop()
				return
			}
		}
		m.previewVP.SetContent(m.promptData.ConfigContent)
		m.previewVP.GotoTop()
		return
	}

	if item.promptFile != nil {
		content := item.promptFile.Content
		if content == "" {
			m.previewVP.SetContent(emptyPreviewStyle.Render("(empty)"))
			m.previewVP.GotoTop()
			return
		}
		if m.mdRenderer != nil {
			rendered, err := m.mdRenderer.Render(content)
			if err == nil {
				m.previewVP.SetContent(strings.TrimSpace(rendered))
				m.previewVP.GotoTop()
				return
			}
		}
		m.previewVP.SetContent(content)
		m.previewVP.GotoTop()
		return
	}

	m.previewVP.SetContent("")
}

// View renders the config browser.
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
		b.WriteString(loadingStyle.Render("Loading config..."))
		return b.String()
	}

	// Calculate pane dimensions.
	explorerWidth := max(m.width*30/100, 20)
	previewWidth := max(m.width-explorerWidth, 20)
	contentHeight := max(m.height-2, 5)

	// Render panes side by side.
	explorer := m.renderExplorer(explorerWidth, contentHeight)
	preview := m.renderPreviewPane(previewWidth, contentHeight)
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, explorer, preview))
	b.WriteString("\n")

	// Status bar / Modal.
	if m.showResetModal {
		b.WriteString(m.renderResetModal())
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

		// Help bar.
		help := helpBarStyle.Render(helpText())
		badge := m.logBadge()
		if badge != "" {
			help = help + "  " + badge
		}
		b.WriteString(help)
	}

	return b.String()
}

// renderExplorer renders the left pane with the list.
func (m Model) renderExplorer(width, height int) string {
	var b strings.Builder

	// Header.
	headerStyle := explorerHeaderStyle
	if m.focusPane == paneExplorer {
		headerStyle = activePaneHeaderStyle
	}
	header := headerStyle.Width(width - 2).Render("Config")
	b.WriteString(header)
	b.WriteString("\n")

	treeHeight := height - 1

	if len(m.items) == 0 {
		empty := lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true).
			Width(width - 2).
			Render("No prompts found")
		return lipgloss.NewStyle().Width(width).Height(height).Render(b.String() + empty)
	}

	// Render items.
	var content strings.Builder
	itemWidth := max(width-5, 10)
	itemStartLines := make([]int, len(m.items))
	currentLine := 0
	for i, item := range m.items {
		itemStartLines[i] = currentLine
		line := m.renderListItem(item, itemWidth)
		var indicator string
		if i == m.cursor && m.focusPane == paneExplorer {
			indicator = " "
			line = selectedStyle.Render(line)
		} else if i == m.cursor {
			indicator = " "
			line = unfocusedSelectedStyle.Render(line)
		} else {
			indicator = " "
		}
		rendered := indicator + line
		content.WriteString(rendered)
		currentLine += lipgloss.Height(rendered)
		if i < len(m.items)-1 {
			content.WriteString("\n")
		}
	}

	// Use viewport for scrolling.
	m.explorerVP.Width = width - 2
	m.explorerVP.Height = treeHeight

	savedOffset := m.explorerVP.YOffset
	m.explorerVP.SetContent(content.String())
	m.explorerVP.SetYOffset(savedOffset)

	// Ensure cursor is visible.
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

// renderListItem renders a single item in the list.
func (m Model) renderListItem(item listItem, width int) string {
	if item.isConfigFile {
		return configItemStyle.Render("cortex.yaml")
	}

	if item.isSectionHeader {
		return sectionHeaderStyle.Render("  " + item.sectionName)
	}

	if item.promptFile != nil {
		connector := treeConnector.Render("  ├─ ")
		filename := item.promptFile.Stage + ".md"
		var badge string
		if item.promptFile.Ejected {
			badge = ejectedBadgeStyle.Render("● ejected")
		} else {
			badge = defaultBadgeStyle.Render("○ default")
		}
		maxName := max(width-20, 5)
		name := truncateToWidth(filename, maxName)
		return connector + name + "  " + badge
	}

	return ""
}

// renderPreviewPane renders the right pane with content preview.
func (m Model) renderPreviewPane(width, height int) string {
	var b strings.Builder

	// Header.
	headerStyle := previewHeaderStyle
	if m.focusPane == panePreview {
		headerStyle = activePaneHeaderStyle
	}

	headerTitle := "Preview"
	item := m.currentItem()
	if item != nil {
		if item.isConfigFile {
			headerTitle = "cortex.yaml"
		} else if item.promptFile != nil {
			headerTitle = item.promptFile.Path
			maxLen := max(width-4, 10)
			if len(headerTitle) > maxLen {
				headerTitle = headerTitle[:maxLen-3] + "..."
			}
		}
	}
	header := headerStyle.Width(width - 2).Render(headerTitle)
	b.WriteString(header)
	b.WriteString("\n")

	previewHeight := max(height-1, 3)

	if item == nil || item.isSectionHeader {
		empty := emptyPreviewStyle.Render("Select an item to preview")
		return lipgloss.NewStyle().Width(width).Height(height).
			Render(b.String() + empty)
	}

	m.previewVP.Width = width - 2
	m.previewVP.Height = previewHeight
	b.WriteString(m.previewVP.View())

	return lipgloss.NewStyle().Width(width).Height(height).Render(b.String())
}

// updateRendererWidth recreates the glamour renderer to match the preview width.
func (m *Model) updateRendererWidth() {
	previewWidth := max(m.width*70/100-4, 40)
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(previewWidth),
	)
	m.mdRenderer = renderer
	if m.promptData != nil {
		m.updatePreview()
	}
}

// loadPrompts returns a command to load all prompts.
func (m Model) loadPrompts() tea.Cmd {
	return func() tea.Msg {
		resp, err := m.client.ListPrompts()
		if err != nil {
			return PromptsErrorMsg{Err: err}
		}
		return PromptsLoadedMsg{Data: resp}
	}
}

// ejectPrompt returns a command to eject a prompt.
func (m Model) ejectPrompt(path string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		_, err := client.EjectPrompt(path)
		return PromptEjectMsg{Err: err}
	}
}

// editPrompt returns a command to edit an ejected prompt.
func (m Model) editPrompt(path string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		err := client.EditPromptInEditor(path)
		return PromptEditMsg{Err: err}
	}
}

// editConfig returns a command to edit cortex.yaml.
func (m Model) editConfig() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		err := client.EditProjectConfigInEditor()
		return ConfigEditMsg{Err: err}
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
