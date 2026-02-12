package dashboard

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
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

// isActive returns true if the project has an active or orphaned architect or ticket session.
func (pd projectData) isActive() bool {
	if pd.architect != nil && (pd.architect.State == "active" || pd.architect.State == "orphaned") {
		return true
	}
	if pd.tickets != nil {
		for _, t := range pd.tickets.Progress {
			if t.HasActiveSession {
				return true
			}
		}
		for _, t := range pd.tickets.Review {
			if t.HasActiveSession {
				return true
			}
		}
	}
	return false
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

	// Unlink confirmation state.
	showUnlinkConfirm bool
	unlinkProjectPath string

	// Architect mode selection state (for orphaned sessions).
	showArchitectModeModal   bool
	architectModeProjectPath string

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

// UnlinkProjectMsg is sent when project unlink completes.
type UnlinkProjectMsg struct {
	ProjectPath string
	Err         error
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

	case UnlinkProjectMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Unlink error: %s", msg.Err)
			m.statusIsError = true
			m.logBuf.Errorf("unlink", "unlink failed: %s: %s", filepath.Base(msg.ProjectPath), msg.Err)
		} else {
			m.statusMsg = "Project unlinked"
			m.statusIsError = false
			m.logBuf.Infof("unlink", "project unlinked: %s", filepath.Base(msg.ProjectPath))
			// Cancel any SSE subscription for this project.
			if cancel, ok := m.sseContexts[msg.ProjectPath]; ok {
				cancel()
				delete(m.sseContexts, msg.ProjectPath)
				delete(m.sseChannels, msg.ProjectPath)
			}
			// Reload projects.
			m.loading = true
			return m, tea.Batch(m.loadProjects(), m.clearStatusAfterDelay())
		}
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
	// Handle unlink confirmation mode first.
	if m.showUnlinkConfirm {
		switch {
		case isKey(msg, KeyYes):
			m.showUnlinkConfirm = false
			path := m.unlinkProjectPath
			m.unlinkProjectPath = ""
			m.statusMsg = "Unlinking project..."
			m.statusIsError = false
			return m, m.unlinkProject(path)
		case isKey(msg, KeyNo, KeyEscape):
			m.showUnlinkConfirm = false
			m.unlinkProjectPath = ""
			m.statusMsg = "Unlink cancelled"
			m.statusIsError = false
			return m, m.clearStatusAfterDelay()
		}
		return m, nil
	}

	// Handle architect mode selection modal.
	if m.showArchitectModeModal {
		return m.handleArchitectModeKey(msg)
	}

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

	// Unlink project.
	if isKey(msg, KeyUnlink) {
		return m.handleUnlinkProject()
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
		if pd.architect != nil && pd.architect.State == "orphaned" {
			m.showArchitectModeModal = true
			m.architectModeProjectPath = pd.project.Path
			return m, nil
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

	// If architect is orphaned, show mode selection modal.
	if pd.architect != nil && pd.architect.State == "orphaned" {
		m.showArchitectModeModal = true
		m.architectModeProjectPath = pd.project.Path
		return m, nil
	}

	m.statusMsg = "Spawning architect..."
	m.statusIsError = false
	return m, m.spawnArchitect(pd.project.Path)
}

// handleUnlinkProject initiates unlink confirmation for the selected project.
func (m Model) handleUnlinkProject() (tea.Model, tea.Cmd) {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return m, nil
	}
	r := m.rows[m.cursor]

	// Only allow unlinking project rows, not session rows.
	if r.kind != rowProject {
		m.statusMsg = "Select a project to unlink"
		m.statusIsError = false
		return m, m.clearStatusAfterDelay()
	}

	pd := m.projects[r.projectIndex]
	m.showUnlinkConfirm = true
	m.unlinkProjectPath = pd.project.Path
	return m, nil
}

// handleArchitectModeKey handles key input for the orphaned architect mode selection modal.
func (m Model) handleArchitectModeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case isKey(msg, KeyRefresh): // 'r' for resume
		m.showArchitectModeModal = false
		path := m.architectModeProjectPath
		m.architectModeProjectPath = ""
		m.statusMsg = "Resuming architect..."
		m.statusIsError = false
		return m, m.spawnArchitectWithMode(path, "resume")
	case isKey(msg, KeyFocus): // 'f' for fresh
		m.showArchitectModeModal = false
		path := m.architectModeProjectPath
		m.architectModeProjectPath = ""
		m.statusMsg = "Starting fresh architect..."
		m.statusIsError = false
		return m, m.spawnArchitectWithMode(path, "fresh")
	case isKey(msg, KeyEscape):
		m.showArchitectModeModal = false
		m.architectModeProjectPath = ""
		m.statusMsg = "Spawn cancelled"
		m.statusIsError = false
		return m, m.clearStatusAfterDelay()
	}
	return m, nil
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

	// Unlink confirmation dialog.
	if m.showUnlinkConfirm {
		title := filepath.Base(m.unlinkProjectPath)
		confirmMsg := fmt.Sprintf("Unlink project '%s'? [y]es [n]o", title)
		b.WriteString(warnBadgeStyle.Render(confirmMsg))
		b.WriteString("\n")
		b.WriteString(mutedStyleRender.Render(m.unlinkProjectPath))
		return b.String()
	}

	// Architect mode selection dialog.
	if m.showArchitectModeModal {
		title := filepath.Base(m.architectModeProjectPath)
		prompt := fmt.Sprintf("Orphaned architect for '%s'", title)
		options := "[r]esume  [f]resh  [esc] cancel"
		b.WriteString(warnBadgeStyle.Render(prompt))
		b.WriteString("\n")
		b.WriteString(helpBarStyle.Render(options))
		return b.String()
	}

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

	// Architect state indicator.
	architectActive := pd.architect != nil && pd.architect.State == "active"
	architectOrphaned := pd.architect != nil && pd.architect.State == "orphaned"
	indicator := "○"
	if architectOrphaned {
		indicator = "◌"
	} else if architectActive {
		indicator = architectSessionIcon(pd.architect)
	}

	// Title.
	title := pd.project.Title
	if title == "" {
		title = filepath.Base(pd.project.Path)
	}

	// Architect status badge.
	archBadge := ""
	if architectOrphaned {
		archBadge = " [arch: orphaned]"
	} else if architectActive && pd.architect.Session != nil {
		archBadge = architectStatusBadge(pd.architect.Session)
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
		plainLine := fmt.Sprintf("%s %s%s %s", indicator, title, archBadge, counts)
		return selectedStyle.Render(plainLine)
	}

	if architectOrphaned {
		return orphanedIconStyle.Render(indicator) + " " + projectStyle.Render(title) + orphanedIconStyle.Render(archBadge) + " " + countsStyle.Render(counts)
	}
	if architectActive {
		return activeIconStyle.Render(indicator) + " " + projectStyle.Render(title) + activeIconStyle.Render(archBadge) + " " + countsStyle.Render(counts)
	}
	return mutedStyleRender.Render(indicator) + " " + dimmedProjectStyle.Render(title) + " " + countsStyle.Render(counts)
}

// architectSessionIcon returns an icon based on the architect's agent status.
func architectSessionIcon(arch *sdk.ArchitectStateResponse) string {
	if arch.Session == nil || arch.Session.Status == nil {
		return "●"
	}
	symbols := map[string]string{
		"starting":           "▶",
		"in_progress":        "●",
		"idle":               "○",
		"waiting_permission": "⏸",
		"error":              "✗",
	}
	if s, ok := symbols[*arch.Session.Status]; ok {
		return s
	}
	return "●"
}

// architectStatusBadge returns a short status badge for the architect session.
func architectStatusBadge(sess *sdk.ArchitectSessionResponse) string {
	dur := formatDuration(time.Since(sess.StartedAt))
	if sess.Tool != nil && *sess.Tool != "" {
		return fmt.Sprintf(" [arch: %s %s]", *sess.Tool, dur)
	}
	return fmt.Sprintf(" [arch: %s]", dur)
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
	if ticket.SessionStartedAt != nil {
		dur = formatDuration(time.Since(*ticket.SessionStartedAt))
	}

	if selected {
		plain := fmt.Sprintf("%s%s %-24s %-10s %s", indent, icon, name, badge, dur)
		return selectedStyle.Render(plain)
	}

	return fmt.Sprintf("%s%s %-24s %-10s %s", indent, styledIcon, sessionStyle.Render(name), badgeStyled, durationStyle.Render(dur))
}

// --- Tree building ---

// rebuildRows flattens the project tree into rows.
func (m *Model) rebuildRows() {
	// Sort projects: active first; among active projects, newest session first.
	slices.SortStableFunc(m.projects, func(a, b projectData) int {
		aActive, bActive := a.isActive(), b.isActive()
		if aActive && !bActive {
			return -1
		}
		if !aActive && bActive {
			return 1
		}
		if aActive && bActive {
			aNewest := newestSessionTime(a)
			bNewest := newestSessionTime(b)
			if aNewest.After(bNewest) {
				return -1
			}
			if bNewest.After(aNewest) {
				return 1
			}
		}
		return 0
	})

	var rows []row
	for i, pd := range m.projects {
		rows = append(rows, row{kind: rowProject, projectIndex: i})

		// Collect all tickets with active sessions from progress and review.
		if pd.tickets != nil {
			var sessionTickets []sdk.TicketSummary
			for _, t := range pd.tickets.Progress {
				if t.HasActiveSession {
					sessionTickets = append(sessionTickets, t)
				}
			}
			for _, t := range pd.tickets.Review {
				if t.HasActiveSession {
					sessionTickets = append(sessionTickets, t)
				}
			}
			// Sort by SessionStartedAt descending (most recent first).
			slices.SortStableFunc(sessionTickets, func(a, b sdk.TicketSummary) int {
				aTime := a.Updated
				if a.SessionStartedAt != nil {
					aTime = *a.SessionStartedAt
				}
				bTime := b.Updated
				if b.SessionStartedAt != nil {
					bTime = *b.SessionStartedAt
				}
				if aTime.After(bTime) {
					return -1
				}
				if bTime.After(aTime) {
					return 1
				}
				return 0
			})
			for _, t := range sessionTickets {
				rows = append(rows, row{kind: rowSession, projectIndex: i, ticketID: t.ID})
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

// newestSessionTime returns the most recent session start time across architect
// and ticket sessions for a project. Returns zero time if no sessions exist.
func newestSessionTime(pd projectData) time.Time {
	var newest time.Time
	if pd.architect != nil && pd.architect.Session != nil {
		newest = pd.architect.Session.StartedAt
	}
	if pd.tickets != nil {
		for _, t := range pd.tickets.Progress {
			if t.SessionStartedAt != nil && t.SessionStartedAt.After(newest) {
				newest = *t.SessionStartedAt
			}
		}
		for _, t := range pd.tickets.Review {
			if t.SessionStartedAt != nil && t.SessionStartedAt.After(newest) {
				newest = *t.SessionStartedAt
			}
		}
	}
	return newest
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

		tickets, err := client.ListAllTickets("", nil, "")
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

// spawnArchitectWithMode spawns an architect with an explicit mode (resume/fresh).
func (m Model) spawnArchitectWithMode(projectPath, mode string) tea.Cmd {
	return func() tea.Msg {
		client := sdk.DefaultClient(projectPath)
		_, err := client.SpawnArchitect(mode)
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

// unlinkProject removes a project from the global registry.
func (m Model) unlinkProject(projectPath string) tea.Cmd {
	return func() tea.Msg {
		err := m.globalClient.UnlinkProject(projectPath)
		return UnlinkProjectMsg{ProjectPath: projectPath, Err: err}
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
