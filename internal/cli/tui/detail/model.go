package detail

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

type TabKind string

const (
	TabKindMarkdown TabKind = "markdown"
	TabKindChanges  TabKind = "changes"
)

type Tab struct {
	Label   string
	Content string
	Kind    TabKind
}

type EditResult struct {
	Title    string
	Subtitle string
	Tabs     []Tab
}

type EditFunc = tea.Cmd
type ChangesLoader = tea.Cmd

type Option func(*Model)

type ChangeFile struct {
	Path      string
	OldPath   string
	Status    string
	IsBinary  bool
	Additions int
	Deletions int
	Patch     string
}

type ChangeCommit struct {
	SHA        string
	Subject    string
	AuthorName string
	AuthoredAt time.Time
	Files      []ChangeFile
}

type ChangesData struct {
	Repo    string
	Commits []ChangeCommit
}

type Model struct {
	title    string
	subtitle string
	tabs     []Tab
	offsets  []int
	active   int

	ticketID  string
	indexPath string
	onEdit    EditFunc
	status    string

	loadChanges    ChangesLoader
	changes        *ChangesData
	changesLoading bool
	changesLoadErr error
	selectedCommit int

	viewport   viewport.Model
	mdRenderer *glamour.TermRenderer
	pendingG   bool

	width, height int
	ready         bool
}

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))

	subtitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("62")).
			Padding(0, 1)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")).
				Padding(0, 1)

	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	emptyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	selectedCommitStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("255")).
				Background(lipgloss.Color("62")).
				Padding(0, 1)

	selectedCommitMetaStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")).
				Background(lipgloss.Color("62")).
				Padding(0, 1)

	commitStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)

	commitMetaStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Padding(0, 1)

	sidebarHintStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Padding(0, 1)

	diffHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255"))

	diffMetaStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	diffRuleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	diffAddedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	diffDeletedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("203"))

	diffHunkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("81")).
			Bold(true)

	diffFileMetaStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244"))

	diffHeaderLineStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("111")).
				Bold(true)
)

type editFinishedMsg struct {
	result EditResult
	err    error
}

type changesLoadedMsg struct {
	changes *ChangesData
	err     error
}

func EditFinished(result EditResult, err error) tea.Msg {
	return editFinishedMsg{result: result, err: err}
}

func ChangesLoaded(changes *ChangesData, err error) tea.Msg {
	return changesLoadedMsg{changes: changes, err: err}
}

func WithEditableTicket(ticketID, indexPath string, onEdit EditFunc) Option {
	return func(m *Model) {
		m.ticketID = ticketID
		m.indexPath = indexPath
		m.onEdit = onEdit
	}
}

func WithChangesLoader(loader ChangesLoader) Option {
	return func(m *Model) {
		m.loadChanges = loader
	}
}

func New(title, subtitle string, tabs []Tab, opts ...Option) Model {
	if len(tabs) == 0 {
		tabs = []Tab{{Label: "Overview", Content: "_No content available._", Kind: TabKindMarkdown}}
	}

	for i := range tabs {
		if tabs[i].Kind == "" {
			tabs[i].Kind = TabKindMarkdown
		}
	}

	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)

	model := Model{
		title:      title,
		subtitle:   subtitle,
		tabs:       tabs,
		offsets:    make([]int, len(tabs)),
		mdRenderer: renderer,
	}

	for _, opt := range opts {
		opt(&model)
	}

	return model
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.updateRendererWidth()
		m.syncViewportSize()
		m.renderActiveTab()
		return m, m.ensureChangesLoaded()

	case editFinishedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Edit failed: %v", msg.err)
			return m, nil
		}

		m.applyEditResult(msg.result)
		m.status = fmt.Sprintf("Reloaded ticket %s", m.ticketID)
		return m, nil

	case changesLoadedMsg:
		m.changesLoading = false
		m.changes = msg.changes
		m.changesLoadErr = msg.err
		if m.selectedCommit >= len(m.changeCommits()) {
			m.selectedCommit = max(len(m.changeCommits())-1, 0)
		}
		m.renderActiveTab()
		return m, nil

	case tea.KeyMsg:
		m.syncViewportSize()

		if m.isChangesTabActive() {
			return m.updateChangesTab(msg)
		}
		return m.updateMarkdownTab(msg)
	}

	return m, nil
}

func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	m.syncViewportSize()

	var b strings.Builder
	b.WriteString(titleStyle.Render(m.title))
	b.WriteString("\n")
	if m.subtitle != "" {
		b.WriteString(subtitleStyle.Render(m.subtitle))
		b.WriteString("\n")
	}
	b.WriteString(m.renderTabBar())
	b.WriteString("\n")
	if m.isChangesTabActive() {
		b.WriteString(m.renderChangesView())
	} else {
		b.WriteString(m.viewport.View())
	}
	b.WriteString("\n")
	if m.status != "" {
		b.WriteString(helpStyle.Render(m.status))
		b.WriteString("\n")
	}
	b.WriteString(helpStyle.Render(m.helpText()))

	return b.String()
}

func (m Model) canEdit() bool {
	return m.ticketID != "" && m.indexPath != "" && m.onEdit != nil
}

func (m *Model) updateChangesTab(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit
	case "tab", "l":
		return m.switchTab(1)
	case "shift+tab", "h":
		return m.switchTab(-1)
	case "e":
		if m.canEdit() {
			m.pendingG = false
			m.status = ""
			return m, m.onEdit
		}
	case "j":
		m.pendingG = false
		m.moveCommit(1)
		return m, nil
	case "k":
		m.pendingG = false
		m.moveCommit(-1)
		return m, nil
	case "G":
		m.pendingG = false
		m.scrollToBottom()
		m.offsets[m.active] = m.viewport.YOffset
		return m, nil
	case "g":
		if m.pendingG {
			m.pendingG = false
			m.viewport.GotoTop()
			m.offsets[m.active] = m.viewport.YOffset
		} else {
			m.pendingG = true
		}
		return m, nil
	}

	m.pendingG = false

	switch msg.String() {
	case "down":
		m.viewport.ScrollDown(1)
	case "up":
		m.viewport.ScrollUp(1)
	case "ctrl+d":
		m.viewport.HalfPageDown()
	case "ctrl+u":
		m.viewport.HalfPageUp()
	}

	m.offsets[m.active] = m.viewport.YOffset
	return m, nil
}

func (m *Model) updateMarkdownTab(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit
	case "tab", "l":
		return m.switchTab(1)
	case "shift+tab", "h":
		return m.switchTab(-1)
	case "G":
		m.pendingG = false
		m.scrollToBottom()
		m.offsets[m.active] = m.viewport.YOffset
		return m, nil
	case "g":
		if m.pendingG {
			m.pendingG = false
			m.viewport.GotoTop()
			m.offsets[m.active] = m.viewport.YOffset
		} else {
			m.pendingG = true
		}
		return m, nil
	case "e":
		if m.canEdit() {
			m.pendingG = false
			m.status = ""
			return m, m.onEdit
		}
	}

	m.pendingG = false

	switch msg.String() {
	case "j", "down":
		m.viewport.ScrollDown(1)
	case "k", "up":
		m.viewport.ScrollUp(1)
	case "ctrl+d":
		m.viewport.HalfPageDown()
	case "ctrl+u":
		m.viewport.HalfPageUp()
	}

	m.offsets[m.active] = m.viewport.YOffset
	return m, nil
}

func (m *Model) applyEditResult(result EditResult) {
	currentLabel := ""
	if len(m.tabs) > 0 && m.active < len(m.tabs) {
		currentLabel = m.tabs[m.active].Label
	}

	m.title = result.Title
	m.subtitle = result.Subtitle
	m.tabs = result.Tabs
	if len(m.tabs) == 0 {
		m.tabs = []Tab{{Label: "Overview", Content: "_No content available._", Kind: TabKindMarkdown}}
	}
	for i := range m.tabs {
		if m.tabs[i].Kind == "" {
			m.tabs[i].Kind = TabKindMarkdown
		}
	}

	m.offsets = make([]int, len(m.tabs))
	m.active = 0
	for i, tab := range m.tabs {
		if tab.Label == currentLabel {
			m.active = i
			break
		}
	}

	m.changes = nil
	m.changesLoading = false
	m.changesLoadErr = nil
	m.selectedCommit = 0
	m.renderActiveTab()
}

func (m *Model) updateRendererWidth() {
	wrapWidth := max(m.width-6, 40)
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(wrapWidth),
	)
	m.mdRenderer = renderer
}

func (m *Model) renderActiveTab() {
	if len(m.tabs) == 0 {
		m.viewport.SetContent(emptyStyle.Render("No content available"))
		m.viewport.SetYOffset(0)
		return
	}

	if m.isChangesTabActive() {
		m.renderChangesContent()
		return
	}

	content := strings.TrimSpace(m.tabs[m.active].Content)
	if content == "" {
		content = "_No content available._"
	}

	rendered := content
	if m.mdRenderer != nil {
		if out, err := m.mdRenderer.Render(content); err == nil {
			rendered = strings.TrimSpace(out)
		}
	}

	m.viewport.SetContent(rendered)
	m.clampYOffset(m.offsets[m.active])
}

func (m *Model) switchTab(delta int) (tea.Model, tea.Cmd) {
	if len(m.tabs) == 0 {
		return m, nil
	}

	m.offsets[m.active] = m.viewport.YOffset
	m.active = (m.active + delta + len(m.tabs)) % len(m.tabs)
	m.pendingG = false
	m.renderActiveTab()
	return m, m.ensureChangesLoaded()
}

func (m *Model) ensureChangesLoaded() tea.Cmd {
	if !m.isChangesTabActive() || m.loadChanges == nil || m.changesLoading || m.changes != nil || m.changesLoadErr != nil {
		return nil
	}
	m.changesLoading = true
	m.renderActiveTab()
	return m.loadChanges
}

func (m *Model) syncViewportSize() {
	if !m.ready {
		return
	}
	m.viewport.Width = max(m.contentWidth(), 20)
	m.viewport.Height = max(m.height-m.headerLineCount(), 3)
	m.clampYOffset(m.viewport.YOffset)
}

func (m Model) headerLineCount() int {
	lines := 3
	if m.subtitle != "" {
		lines++
	}
	if m.status != "" {
		lines++
	}
	return lines
}

func (m *Model) clampYOffset(offset int) {
	maxOffset := max(m.viewport.TotalLineCount()-m.viewport.Height, 0)
	if offset < 0 {
		offset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	m.viewport.SetYOffset(offset)
}

func (m *Model) scrollToBottom() {
	m.clampYOffset(m.viewport.TotalLineCount() - m.viewport.Height)
}

func (m Model) renderTabBar() string {
	parts := make([]string, 0, len(m.tabs))
	for i, tab := range m.tabs {
		label := fmt.Sprintf(" %s ", tab.Label)
		if i == m.active {
			parts = append(parts, activeTabStyle.Render(label))
			continue
		}
		parts = append(parts, inactiveTabStyle.Render(label))
	}
	return strings.Join(parts, "")
}

func (m Model) helpText() string {
	if m.isChangesTabActive() {
		parts := []string{"tab/h/l tabs", "j/k commits", "↑/↓ diff", "ctrl+d/u page", "gg/G jump"}
		if m.canEdit() {
			parts = append(parts, "e edit")
		}
		parts = append(parts, "q close")
		return strings.Join(parts, "  ")
	}

	parts := []string{"tab/h/l tabs", "j/k scroll", "ctrl+d/u page", "gg/G jump"}
	if m.canEdit() {
		parts = append(parts, "e edit")
	}
	parts = append(parts, "q close")
	return strings.Join(parts, "  ")
}

func (m Model) contentWidth() int {
	if m.isChangesTabActive() {
		_, diffWidth := m.paneWidths()
		return diffWidth - 1
	}
	return m.width - 2
}

func (m Model) paneWidths() (int, int) {
	total := max(m.width-2, 20)
	sidebar := max(total/4, 24)
	if sidebar > total-20 {
		sidebar = max(total/3, 12)
	}
	diff := max(total-sidebar-1, 20)
	if sidebar+1+diff > total {
		sidebar = max(total-diff-1, 12)
	}
	return sidebar, diff
}

func (m Model) activeTab() Tab {
	if len(m.tabs) == 0 || m.active >= len(m.tabs) {
		return Tab{Label: "Overview", Content: "_No content available._", Kind: TabKindMarkdown}
	}
	tab := m.tabs[m.active]
	if tab.Kind == "" {
		tab.Kind = TabKindMarkdown
	}
	return tab
}

func (m Model) isChangesTabActive() bool {
	return m.activeTab().Kind == TabKindChanges
}

func (m Model) changeCommits() []ChangeCommit {
	if m.changes == nil {
		return nil
	}
	return m.changes.Commits
}

func (m *Model) moveCommit(delta int) {
	commits := m.changeCommits()
	if len(commits) == 0 {
		return
	}
	next := m.selectedCommit + delta
	if next < 0 {
		next = 0
	}
	if next >= len(commits) {
		next = len(commits) - 1
	}
	if next == m.selectedCommit {
		return
	}
	m.selectedCommit = next
	m.offsets[m.active] = 0
	m.renderChangesContent()
}

func (m *Model) renderChangesContent() {
	switch {
	case m.changesLoading:
		m.viewport.SetContent(emptyStyle.Render("Loading changes..."))
		m.viewport.SetYOffset(0)
	case m.changesLoadErr != nil:
		m.viewport.SetContent(emptyStyle.Render(fmt.Sprintf("Unable to load changes.\n\n%s", m.changesLoadErr.Error())))
		m.viewport.SetYOffset(0)
	case m.changes == nil || len(m.changes.Commits) == 0:
		m.viewport.SetContent(emptyStyle.Render("No commit diffs available"))
		m.viewport.SetYOffset(0)
	default:
		commits := m.changeCommits()
		if m.selectedCommit >= len(commits) {
			m.selectedCommit = len(commits) - 1
		}
		if m.selectedCommit < 0 {
			m.selectedCommit = 0
		}
		m.viewport.SetContent(m.renderCommitDiff(commits[m.selectedCommit]))
		m.clampYOffset(m.offsets[m.active])
	}
}

func (m Model) renderChangesView() string {
	sidebarWidth, diffWidth := m.paneWidths()
	sidebar := lipgloss.NewStyle().
		Width(sidebarWidth).
		Height(m.viewport.Height).
		BorderRight(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("239")).
		Render(m.renderCommitSidebar(sidebarWidth))
	diff := lipgloss.NewStyle().
		Width(diffWidth).
		Height(m.viewport.Height).
		PaddingLeft(1).
		Render(m.viewport.View())
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, diff)
}

func (m Model) renderCommitSidebar(width int) string {
	innerWidth := max(width-2, 10)
	height := max(m.viewport.Height, 3)

	if m.changesLoading {
		return lipgloss.NewStyle().Width(innerWidth).Height(height).Render(emptyStyle.Render("Loading commits..."))
	}
	if m.changesLoadErr != nil {
		return lipgloss.NewStyle().Width(innerWidth).Height(height).Render(emptyStyle.Render("Unable to load commits"))
	}

	commits := m.changeCommits()
	if len(commits) == 0 {
		return lipgloss.NewStyle().Width(innerWidth).Height(height).Render(emptyStyle.Render("No commits"))
	}

	hint := sidebarHintStyle.Width(innerWidth).Render("j/k select commit")
	available := max(height-1, 1)
	rowsPerCommit := 3
	visibleCount := max(available/rowsPerCommit, 1)
	start := 0
	if m.selectedCommit >= visibleCount {
		start = m.selectedCommit - visibleCount + 1
	}
	if start+visibleCount > len(commits) {
		start = max(len(commits)-visibleCount, 0)
	}

	lines := make([]string, 0, height)
	for i := start; i < len(commits) && i < start+visibleCount; i++ {
		commit := commits[i]
		selected := i == m.selectedCommit
		subject := truncate(commit.Subject, max(innerWidth-10, 8))
		meta := truncate(formatCommitMeta(commit), max(innerWidth-2, 8))
		title := fmt.Sprintf("%s %s", shortSHA(commit.SHA), subject)
		if selected {
			lines = append(lines, selectedCommitStyle.Width(innerWidth).Render(title))
			lines = append(lines, selectedCommitMetaStyle.Width(innerWidth).Render(meta))
		} else {
			lines = append(lines, commitStyle.Width(innerWidth).Render(title))
			lines = append(lines, commitMetaStyle.Width(innerWidth).Render(meta))
		}
		lines = append(lines, "")
	}

	if len(lines) > available {
		lines = lines[:available]
	}
	for len(lines) < available {
		lines = append(lines, "")
	}
	lines = append(lines, hint)
	return strings.Join(lines, "\n")
}

func (m Model) renderCommitDiff(commit ChangeCommit) string {
	if len(commit.Files) == 0 {
		return emptyStyle.Render("No file diffs for this commit")
	}

	var b strings.Builder
	for i, file := range commit.Files {
		b.WriteString(diffHeaderStyle.Render(formatFileHeader(file)))
		b.WriteString("\n")
		b.WriteString(diffMetaStyle.Render(fmt.Sprintf("%s  (+%d -%d)", formatFileStatus(file), file.Additions, file.Deletions)))
		b.WriteString("\n")
		b.WriteString(diffRuleStyle.Render(strings.Repeat("─", max(m.viewport.Width-2, 10))))
		b.WriteString("\n")

		patch := strings.TrimRight(file.Patch, "\n")
		if file.IsBinary && patch == "" {
			patch = "Binary file"
		}
		if patch == "" {
			patch = "(no patch available)"
		}
		b.WriteString(colorizePatch(patch))

		if i < len(commit.Files)-1 {
			b.WriteString("\n\n")
		}
	}
	return b.String()
}

func formatCommitMeta(commit ChangeCommit) string {
	name := commit.AuthorName
	if name == "" {
		name = "Unknown"
	}
	date := "-"
	if !commit.AuthoredAt.IsZero() {
		date = commit.AuthoredAt.Local().Format("Jan 2")
	}
	return fmt.Sprintf("%s · %s", name, date)
}

func formatFileHeader(file ChangeFile) string {
	if file.Status == "renamed" && file.OldPath != "" && file.OldPath != file.Path {
		return fmt.Sprintf("%s -> %s", file.OldPath, file.Path)
	}
	return file.Path
}

func formatFileStatus(file ChangeFile) string {
	if file.Status == "" {
		return "modified"
	}
	return file.Status
}

func shortSHA(sha string) string {
	if len(sha) <= 8 {
		return sha
	}
	return sha[:8]
}

func truncate(value string, width int) string {
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	if width <= 1 {
		return string(runes[:width])
	}
	return string(runes[:width-1]) + "…"
}

func colorizePatch(patch string) string {
	lines := strings.Split(patch, "\n")
	for i, line := range lines {
		lines[i] = stylePatchLine(line)
	}
	return strings.Join(lines, "\n")
}

func stylePatchLine(line string) string {
	switch {
	case strings.HasPrefix(line, "+++ "), strings.HasPrefix(line, "--- "), strings.HasPrefix(line, "index "), strings.HasPrefix(line, "new file mode "), strings.HasPrefix(line, "deleted file mode "), strings.HasPrefix(line, "similarity index "), strings.HasPrefix(line, "rename from "), strings.HasPrefix(line, "rename to "):
		return diffFileMetaStyle.Render(line)
	case strings.HasPrefix(line, "diff --git "):
		return diffHeaderLineStyle.Render(line)
	case strings.HasPrefix(line, "@@"):
		return diffHunkStyle.Render(line)
	case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++ "):
		return diffAddedStyle.Render(line)
	case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "--- "):
		return diffDeletedStyle.Render(line)
	default:
		return line
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
