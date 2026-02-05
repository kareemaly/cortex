package dashboard

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/cli/tui/tuilog"
)

// rowKind identifies what a tree row represents.
type rowKind int

const (
	rowProject rowKind = iota
	rowSession
)

// row is a flattened tree entry for cursor navigation.
type row struct {
	kind         rowKind
	projectIndex int    // index into projects slice
	ticketID     string // non-empty for ticket session rows
}

// projectData holds per-project state.
type projectData struct {
	project   sdk.ProjectResponse
	tickets   *sdk.ListAllTicketsResponse
	architect *sdk.ArchitectStateResponse
	loading   bool
	err       error
}

// Model is the main Bubbletea model for the dashboard.
type Model struct {
	globalClient *sdk.Client
	projects     []projectData
	rows         []row
	cursor       int
	scrollOffset int

	// SSE subscriptions per project path.
	sseContexts map[string]context.CancelFunc
	sseChannels map[string]<-chan sdk.Event

	width, height int
	ready         bool
	loading       bool
	err           error
	statusMsg     string
	statusIsError bool

	// Vim navigation state.
	pendingG bool

	// Log viewer state.
	logBuf        *tuilog.Buffer
	logViewer     tuilog.Viewer
	showLogViewer bool
}

// --- Message types ---

// ProjectsLoadedMsg is sent when the project list is fetched.
type ProjectsLoadedMsg struct {
	Projects []sdk.ProjectResponse
}

// ProjectsErrorMsg is sent when fetching projects fails.
type ProjectsErrorMsg struct {
	Err error
}

// ProjectDetailLoadedMsg is sent when a project's detail data is loaded.
type ProjectDetailLoadedMsg struct {
	ProjectPath string
	Tickets     *sdk.ListAllTicketsResponse
	Architect   *sdk.ArchitectStateResponse
	Err         error
}

// SSEConnectedMsg is sent when an SSE subscription is established.
type SSEConnectedMsg struct {
	ProjectPath string
	Ch          <-chan sdk.Event
	Cancel      context.CancelFunc
}

// SSEEventMsg is sent when an SSE event is received for a project.
type SSEEventMsg struct {
	ProjectPath string
}

// SpawnArchitectMsg is sent when architect spawn completes.
type SpawnArchitectMsg struct {
	ProjectPath string
	Err         error
}

// FocusSuccessMsg is sent when a window is focused.
type FocusSuccessMsg struct {
	Name string
}

// FocusErrorMsg is sent when focusing fails.
type FocusErrorMsg struct {
	Err error
}

// ClearStatusMsg clears the status bar.
type ClearStatusMsg struct{}

// TickMsg is sent for duration display refresh.
type TickMsg struct{}

// New creates a new dashboard model.
func New(client *sdk.Client, logBuf *tuilog.Buffer) Model {
	return Model{
		globalClient: client,
		loading:      true,
		sseContexts:  make(map[string]context.CancelFunc),
		sseChannels:  make(map[string]<-chan sdk.Event),
		logBuf:       logBuf,
		logViewer:    tuilog.NewViewer(logBuf),
	}
}

// Init starts loading data.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadProjects(), m.tickDuration())
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

	case ProjectsLoadedMsg:
		m.loading = false
		m.err = nil
		m.projects = make([]projectData, len(msg.Projects))
		var cmds []tea.Cmd
		for i, p := range msg.Projects {
			m.projects[i] = projectData{project: p, loading: true}
			if p.Exists {
				cmds = append(cmds, m.loadProjectDetail(p.Path))
				cmds = append(cmds, m.subscribeProjectEvents(p.Path))
			}
		}
		m.rebuildRows()
		m.logBuf.Debugf("api", "projects loaded: %d", len(msg.Projects))
		return m, tea.Batch(cmds...)

	case ProjectsErrorMsg:
		m.loading = false
		m.err = msg.Err
		m.logBuf.Errorf("api", "failed to load projects: %s", msg.Err)
		return m, nil

	case ProjectDetailLoadedMsg:
		idx := m.findProject(msg.ProjectPath)
		if idx < 0 {
			return m, nil
		}
		m.projects[idx].loading = false
		if msg.Err != nil {
			m.projects[idx].err = msg.Err
			m.logBuf.Errorf("api", "failed to load project detail: %s: %s", filepath.Base(msg.ProjectPath), msg.Err)
		} else {
			m.projects[idx].tickets = msg.Tickets
			m.projects[idx].architect = msg.Architect
			m.projects[idx].err = nil
			m.logBuf.Debugf("api", "project detail loaded: %s", filepath.Base(msg.ProjectPath))
		}
		m.rebuildRows()
		return m, nil

	case SSEConnectedMsg:
		m.sseContexts[msg.ProjectPath] = msg.Cancel
		m.sseChannels[msg.ProjectPath] = msg.Ch
		m.logBuf.Infof("sse", "connected: %s", filepath.Base(msg.ProjectPath))
		return m, m.waitForProjectEvent(msg.ProjectPath)

	case SSEEventMsg:
		m.logBuf.Debugf("sse", "event: %s", filepath.Base(msg.ProjectPath))
		// Reload this project's data and wait for next event.
		idx := m.findProject(msg.ProjectPath)
		cmds := []tea.Cmd{m.waitForProjectEvent(msg.ProjectPath)}
		if idx >= 0 {
			cmds = append(cmds, m.loadProjectDetail(msg.ProjectPath))
		}
		return m, tea.Batch(cmds...)

	case SpawnArchitectMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Spawn error: %s", msg.Err)
			m.statusIsError = true
			m.logBuf.Errorf("spawn", "architect spawn failed: %s: %s", filepath.Base(msg.ProjectPath), msg.Err)
		} else {
			m.statusMsg = "Architect spawned"
			m.statusIsError = false
			m.logBuf.Infof("spawn", "architect spawned: %s", filepath.Base(msg.ProjectPath))
			// Reload project detail.
			idx := m.findProject(msg.ProjectPath)
			if idx >= 0 {
				return m, tea.Batch(m.loadProjectDetail(msg.ProjectPath), m.clearStatusAfterDelay())
			}
		}
		return m, m.clearStatusAfterDelay()

	case FocusSuccessMsg:
		m.statusMsg = fmt.Sprintf("Focused: %s", msg.Name)
		m.statusIsError = false
		m.logBuf.Infof("focus", "focused: %s", msg.Name)
		return m, m.clearStatusAfterDelay()

	case FocusErrorMsg:
		m.statusMsg = fmt.Sprintf("Focus error: %s", msg.Err)
		m.statusIsError = true
		m.logBuf.Errorf("focus", "focus failed: %s", msg.Err)
		return m, m.clearStatusAfterDelay()

	case ClearStatusMsg:
		m.statusMsg = ""
		m.statusIsError = false
		return m, nil

	case TickMsg:
		return m, m.tickDuration()
	}

	return m, nil
}

// handleKeyMsg processes keyboard input.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Quit.
	if isKey(msg, KeyQuit, KeyCtrlC) {
		for _, cancel := range m.sseContexts {
			cancel()
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

	// Don't process other keys while loading.
	if m.loading {
		return m, nil
	}

	// If there's an error, only allow refresh.
	if m.err != nil {
		if isKey(msg, KeyRefresh) {
			m.loading = true
			m.err = nil
			return m, m.loadProjects()
		}
		return m, nil
	}

	// Handle 'G' - jump to last.
	if isKey(msg, KeyShiftG) {
		m.pendingG = false
		if len(m.rows) > 0 {
			m.cursor = len(m.rows) - 1
		}
		return m, nil
	}

	// Handle 'g' key for 'gg' sequence.
	if isKey(msg, KeyG) {
		if m.pendingG {
			m.pendingG = false
			m.cursor = 0
			m.scrollOffset = 0
		} else {
			m.pendingG = true
		}
		return m, nil
	}

	// Clear pending g on any other key.
	m.pendingG = false

	// Scroll up (ctrl+u).
	if isKey(msg, KeyCtrlU) {
		m.cursor = max(m.cursor-10, 0)
		return m, nil
	}

	// Scroll down (ctrl+d).
	if isKey(msg, KeyCtrlD) {
		if len(m.rows) > 0 {
			m.cursor = min(m.cursor+10, len(m.rows)-1)
		}
		return m, nil
	}

	// Navigate up.
	if isKey(msg, KeyUp, KeyK) {
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	}

	// Navigate down.
	if isKey(msg, KeyDown, KeyJ) {
		if m.cursor < len(m.rows)-1 {
			m.cursor++
		}
		return m, nil
	}

	// Focus current row.
	if isKey(msg, KeyEnter, KeyL, KeyFocus) {
		return m.handleFocusCurrentRow()
	}

	// Spawn architect.
	if isKey(msg, KeySpawn) {
		return m.handleSpawnArchitect()
	}

	// Refresh.
	if isKey(msg, KeyRefresh) {
		m.loading = true
		// Cancel all SSE subscriptions.
		for path, cancel := range m.sseContexts {
			cancel()
			delete(m.sseContexts, path)
			delete(m.sseChannels, path)
		}
		// Reset project state.
		for i := range m.projects {
			m.projects[i].tickets = nil
			m.projects[i].architect = nil
			m.projects[i].loading = false
			m.projects[i].err = nil
		}
		return m, m.loadProjects()
	}

	return m, nil
}

// handleFocusCurrentRow focuses the architect (project row) or ticket session (session row).
func (m Model) handleFocusCurrentRow() (tea.Model, tea.Cmd) {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return m, nil
	}
	r := m.rows[m.cursor]
	pd := m.projects[r.projectIndex]

	if r.kind == rowProject {
		if !pd.project.Exists {
			m.statusMsg = "Project is stale"
			m.statusIsError = false
			return m, m.clearStatusAfterDelay()
		}
		if pd.architect != nil && pd.architect.State == "active" {
			// Focus the active architect.
			m.statusMsg = "Focusing architect..."
			m.statusIsError = false
			return m, m.spawnArchitect(pd.project.Path)
		}
		m.statusMsg = "No active architect. Press [s] to spawn."
		m.statusIsError = false
		return m, m.clearStatusAfterDelay()
	}

	// Session row — focus ticket.
	m.statusMsg = "Focusing session..."
	m.statusIsError = false
	return m, m.focusTicket(pd.project.Path, r.ticketID)
}

// handleSpawnArchitect spawns an architect for the selected project.
func (m Model) handleSpawnArchitect() (tea.Model, tea.Cmd) {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return m, nil
	}
	r := m.rows[m.cursor]

	pd := m.projects[r.projectIndex]
	if !pd.project.Exists {
		m.statusMsg = "Project is stale"
		m.statusIsError = false
		return m, m.clearStatusAfterDelay()
	}

	m.statusMsg = "Spawning architect..."
	m.statusIsError = false
	return m, m.spawnArchitect(pd.project.Path)
}

// --- View ---

// View renders the dashboard.
func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	// Log viewer overlay.
	if m.showLogViewer {
		return m.logViewer.View()
	}

	var b strings.Builder

	// Header.
	headerLeft := headerStyle.Render("Cortex Dashboard")
	headerPadding := max(m.width-lipgloss.Width(headerLeft), 0)
	header := headerLeft + strings.Repeat(" ", headerPadding)
	b.WriteString(header)
	b.WriteString("\n\n")

	// Handle error state.
	if m.err != nil {
		errMsg := errorStatusStyle.Render(fmt.Sprintf("Error: %s", m.err))
		b.WriteString(errMsg)
		b.WriteString("\n\n")
		b.WriteString("Press [r] to retry or [q] to quit\n")
		if strings.Contains(m.err.Error(), "connect") {
			b.WriteString("\nIs the daemon running? Start it with: cortexd\n")
		}
		return b.String()
	}

	// Handle loading state.
	if m.loading {
		b.WriteString(loadingStyle.Render("Loading projects..."))
		return b.String()
	}

	// Empty state.
	if len(m.projects) == 0 {
		b.WriteString(loadingStyle.Render("No projects registered. Use 'cortex init' in a project directory."))
		b.WriteString("\n\n")
		b.WriteString(helpBarStyle.Render("[r]efresh  [q]uit"))
		return b.String()
	}

	// Calculate available height for tree.
	// Header (1) + blank (1) + status bar (1) + help bar (1) + margin (1) = ~5 overhead.
	treeHeight := max(m.height-5, 3)

	// Ensure cursor is in visible range.
	m.ensureCursorVisible(treeHeight)

	// Render visible rows.
	endIdx := min(m.scrollOffset+treeHeight, len(m.rows))

	// Top scroll indicator.
	if m.scrollOffset > 0 {
		b.WriteString(mutedStyleRender.Render("▲"))
		b.WriteString("\n")
		treeHeight--
		endIdx = min(m.scrollOffset+treeHeight, len(m.rows))
	}

	for i := m.scrollOffset; i < endIdx; i++ {
		r := m.rows[i]
		selected := i == m.cursor

		switch r.kind {
		case rowProject:
			b.WriteString(m.renderProjectRow(r.projectIndex, selected))
		case rowSession:
			b.WriteString(m.renderSessionRow(r, selected))
		}

		if i < endIdx-1 {
			b.WriteString("\n")
		}
	}

	// Bottom scroll indicator.
	if endIdx < len(m.rows) {
		b.WriteString("\n")
		b.WriteString(mutedStyleRender.Render("▼"))
	}

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

// renderProjectRow renders a single project row.
func (m Model) renderProjectRow(projectIdx int, selected bool) string {
	pd := m.projects[projectIdx]

	// Architect active indicator.
	architectActive := pd.architect != nil && pd.architect.State == "active"
	indicator := "○"
	if architectActive {
		indicator = "●"
	}

	// Title.
	title := pd.project.Title
	if title == "" {
		title = filepath.Base(pd.project.Path)
	}

	// Counts.
	counts := ""
	if pd.project.Counts != nil {
		c := pd.project.Counts
		counts = fmt.Sprintf("(%d backlog · %d prog · %d review)", c.Backlog, c.Progress, c.Review)
	}

	// Loading indicator.
	if pd.loading {
		counts = "(loading...)"
	}

	// Error indicator.
	if pd.err != nil {
		counts = fmt.Sprintf("(error: %s)", pd.err)
	}

	// Stale indicator.
	if !pd.project.Exists {
		line := fmt.Sprintf("%s %s (stale)", indicator, title)
		if selected {
			return selectedStyle.Render(line)
		}
		return staleStyle.Render(line)
	}

	if selected {
		plainLine := fmt.Sprintf("%s %s %s", indicator, title, counts)
		return selectedStyle.Render(plainLine)
	}

	if architectActive {
		return activeIconStyle.Render(indicator) + " " + projectStyle.Render(title) + " " + countsStyle.Render(counts)
	}
	return mutedStyleRender.Render(indicator) + " " + dimmedProjectStyle.Render(title) + " " + countsStyle.Render(counts)
}

// renderSessionRow renders a single session row.
func (m Model) renderSessionRow(r row, selected bool) string {
	pd := m.projects[r.projectIndex]
	indent := "    "

	// Ticket session row.
	ticket := m.findTicket(pd, r.ticketID)
	if ticket == nil {
		return indent + "???"
	}

	icon := agentStatusIcon(*ticket)
	styledIcon := activeIconStyle.Render(icon)
	if ticket.IsOrphaned {
		styledIcon = orphanedIconStyle.Render(icon)
	}
	name := ticket.Title
	if len(name) > 24 {
		name = name[:21] + "..."
	}

	badge := ticket.Status
	if ticket.IsOrphaned {
		badge = "orphaned"
	}
	badgeStyled := progressBadgeStyle.Render(badge)
	if ticket.Status == "review" && !ticket.IsOrphaned {
		badgeStyled = reviewBadgeStyle.Render(badge)
	} else if ticket.IsOrphaned {
		badgeStyled = orphanedIconStyle.Render(badge)
	}

	dur := formatDuration(time.Since(ticket.Updated))

	if selected {
		plain := fmt.Sprintf("%s%s %-24s %-10s %s", indent, icon, name, badge, dur)
		return selectedStyle.Render(plain)
	}

	return fmt.Sprintf("%s%s %-24s %-10s %s", indent, styledIcon, sessionStyle.Render(name), badgeStyled, durationStyle.Render(dur))
}

// --- Tree building ---

// rebuildRows flattens the project tree into rows.
func (m *Model) rebuildRows() {
	var rows []row
	for i, pd := range m.projects {
		rows = append(rows, row{kind: rowProject, projectIndex: i})

		// Add ticket session rows for progress/review tickets with active sessions.
		if pd.tickets != nil {
			for _, t := range pd.tickets.Progress {
				if t.HasActiveSession {
					rows = append(rows, row{kind: rowSession, projectIndex: i, ticketID: t.ID})
				}
			}
			for _, t := range pd.tickets.Review {
				if t.HasActiveSession {
					rows = append(rows, row{kind: rowSession, projectIndex: i, ticketID: t.ID})
				}
			}
		}
	}
	m.rows = rows

	// Clamp cursor.
	if len(m.rows) > 0 {
		if m.cursor >= len(m.rows) {
			m.cursor = len(m.rows) - 1
		}
	} else {
		m.cursor = 0
	}
}

// --- Helpers ---

// findProject returns the index of a project by path, or -1.
func (m Model) findProject(path string) int {
	for i, pd := range m.projects {
		if pd.project.Path == path {
			return i
		}
	}
	return -1
}

// findTicket searches for a ticket by ID in a project's loaded data.
func (m Model) findTicket(pd projectData, ticketID string) *sdk.TicketSummary {
	if pd.tickets == nil {
		return nil
	}
	for i := range pd.tickets.Progress {
		if pd.tickets.Progress[i].ID == ticketID {
			return &pd.tickets.Progress[i]
		}
	}
	for i := range pd.tickets.Review {
		if pd.tickets.Review[i].ID == ticketID {
			return &pd.tickets.Review[i]
		}
	}
	return nil
}

// ensureCursorVisible adjusts scrollOffset for viewport.
func (m *Model) ensureCursorVisible(viewHeight int) {
	if viewHeight <= 0 {
		return
	}
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}
	if m.cursor >= m.scrollOffset+viewHeight {
		m.scrollOffset = m.cursor - viewHeight + 1
	}
}

// agentStatusIcon returns an icon for a ticket's agent status.
func agentStatusIcon(t sdk.TicketSummary) string {
	// Orphaned sessions get a distinct icon.
	if t.IsOrphaned {
		return "◌"
	}

	if t.AgentStatus == nil {
		return "●"
	}
	symbols := map[string]string{
		"starting":           "▶",
		"in_progress":        "●",
		"idle":               "○",
		"waiting_permission": "⏸",
		"error":              "✗",
	}
	if s, ok := symbols[*t.AgentStatus]; ok {
		return s
	}
	return "●"
}

// formatDuration formats a duration into a human-friendly string.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "<1m"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	if hours >= 24 {
		days := hours / 24
		hours = hours % 24
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh %dm", hours, mins)
}

// --- Commands ---

// loadProjects fetches the project list.
func (m Model) loadProjects() tea.Cmd {
	return func() tea.Msg {
		resp, err := m.globalClient.ListProjects()
		if err != nil {
			return ProjectsErrorMsg{Err: err}
		}
		return ProjectsLoadedMsg{Projects: resp.Projects}
	}
}

// loadProjectDetail fetches tickets and architect state for a project.
func (m Model) loadProjectDetail(projectPath string) tea.Cmd {
	return func() tea.Msg {
		client := sdk.DefaultClient(projectPath)

		tickets, err := client.ListAllTickets("", nil)
		if err != nil {
			return ProjectDetailLoadedMsg{ProjectPath: projectPath, Err: err}
		}

		// Architect state is non-fatal.
		architect, _ := client.GetArchitect()

		return ProjectDetailLoadedMsg{
			ProjectPath: projectPath,
			Tickets:     tickets,
			Architect:   architect,
		}
	}
}

// subscribeProjectEvents opens an SSE connection for a project.
func (m Model) subscribeProjectEvents(projectPath string) tea.Cmd {
	return func() tea.Msg {
		client := sdk.DefaultClient(projectPath)
		ctx, cancel := context.WithCancel(context.Background())
		ch, err := client.SubscribeEvents(ctx)
		if err != nil {
			cancel()
			return nil // graceful degradation
		}
		return SSEConnectedMsg{ProjectPath: projectPath, Ch: ch, Cancel: cancel}
	}
}

// waitForProjectEvent waits for the next event on a project's SSE channel.
func (m Model) waitForProjectEvent(projectPath string) tea.Cmd {
	ch, ok := m.sseChannels[projectPath]
	if !ok || ch == nil {
		return nil
	}
	return func() tea.Msg {
		_, ok := <-ch
		if !ok {
			return nil
		}
		return SSEEventMsg{ProjectPath: projectPath}
	}
}

// spawnArchitect spawns an architect for a project.
func (m Model) spawnArchitect(projectPath string) tea.Cmd {
	return func() tea.Msg {
		client := sdk.DefaultClient(projectPath)
		_, err := client.SpawnArchitect("")
		return SpawnArchitectMsg{ProjectPath: projectPath, Err: err}
	}
}

// focusTicket focuses a ticket's tmux window.
func (m Model) focusTicket(projectPath, ticketID string) tea.Cmd {
	return func() tea.Msg {
		client := sdk.DefaultClient(projectPath)
		if err := client.FocusTicket(ticketID); err != nil {
			return FocusErrorMsg{Err: err}
		}
		return FocusSuccessMsg{Name: ticketID[:8]}
	}
}

// clearStatusAfterDelay clears the status message after 3 seconds.
func (m Model) clearStatusAfterDelay() tea.Cmd {
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return ClearStatusMsg{}
	})
}

// tickDuration sends a tick every 30 seconds for duration display refresh.
func (m Model) tickDuration() tea.Cmd {
	return tea.Tick(30*time.Second, func(time.Time) tea.Msg {
		return TickMsg{}
	})
}
