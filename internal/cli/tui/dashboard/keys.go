package dashboard

import (
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type Key string

const (
	KeyQuit    Key = "q"
	KeyUp      Key = "up"
	KeyDown    Key = "down"
	KeyK       Key = "k"
	KeyJ       Key = "j"
	KeyL       Key = "l"
	KeyEnter   Key = "enter"
	KeyFocus   Key = "f"
	KeySpawn   Key = "s"
	KeyRefresh Key = "r"
	KeyCtrlC   Key = "ctrl+c"
	KeyCtrlU   Key = "ctrl+u"
	KeyCtrlD   Key = "ctrl+d"
	KeyG       Key = "g"
	KeyShiftG  Key = "G"
	KeyExclaim Key = "!"
	KeyUnlink  Key = "u"
	KeyKill    Key = "x"
	KeyYes     Key = "y"
	KeyNo      Key = "n"
	KeyEscape  Key = "esc"
)

func isKey(msg tea.KeyMsg, keys ...Key) bool {
	for _, k := range keys {
		if msg.String() == string(k) {
			return true
		}
	}
	return false
}

func helpText() string {
	return "[enter/f] focus  [s]pawn architect  [x] kill  [u]nlink  [r]efresh  [j/k/gg/G] navigate  [!] logs  [q]uit"
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

	if m.showKillConfirm {
		switch {
		case isKey(msg, KeyYes):
			m.showKillConfirm = false
			m.killing = true
			m.statusMsg = "Killing session..."
			m.statusIsError = false
			return m, m.killSession(m.killProjectPath, m.killSessionID)
		case isKey(msg, KeyNo, KeyEscape):
			m.showKillConfirm = false
			m.killProjectPath = ""
			m.killSessionID = ""
			m.killSessionName = ""
			m.statusMsg = "Kill cancelled"
			m.statusIsError = false
			return m, m.clearStatusAfterDelay()
		}
		return m, nil
	}

	if m.showArchitectModeModal {
		return m.handleArchitectModeKey(msg)
	}

	if isKey(msg, KeyQuit, KeyCtrlC) {
		for _, cancel := range m.sseContexts {
			cancel()
		}
		return m, tea.Quit
	}

	if isKey(msg, KeyExclaim) {
		m.showLogViewer = !m.showLogViewer
		if m.showLogViewer {
			m.logViewer.SetSize(m.width, m.height)
			m.logViewer.Reset()
		}
		return m, nil
	}

	if m.loading || m.killing {
		return m, nil
	}

	if m.err != nil {
		if isKey(msg, KeyRefresh) {
			m.loading = true
			m.err = nil
			return m, m.loadProjects()
		}
		return m, nil
	}

	if isKey(msg, KeyShiftG) {
		m.pendingG = false
		if len(m.rows) > 0 {
			m.cursor = len(m.rows) - 1
		}
		return m, nil
	}

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

	m.pendingG = false

	if isKey(msg, KeyCtrlU) {
		m.cursor = max(m.cursor-10, 0)
		return m, nil
	}

	if isKey(msg, KeyCtrlD) {
		if len(m.rows) > 0 {
			m.cursor = min(m.cursor+10, len(m.rows)-1)
		}
		return m, nil
	}

	if isKey(msg, KeyUp, KeyK) {
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	}

	if isKey(msg, KeyDown, KeyJ) {
		if m.cursor < len(m.rows)-1 {
			m.cursor++
		}
		return m, nil
	}

	if isKey(msg, KeyEnter, KeyL, KeyFocus) {
		return m.handleFocusCurrentRow()
	}

	if isKey(msg, KeySpawn) {
		return m.handleSpawnArchitect()
	}

	if isKey(msg, KeyKill) {
		return m.handleKillSession()
	}

	if isKey(msg, KeyUnlink) {
		return m.handleUnlinkArchitect()
	}

	if isKey(msg, KeyRefresh) {
		m.loading = true
		for path, cancel := range m.sseContexts {
			cancel()
			delete(m.sseContexts, path)
			delete(m.sseChannels, path)
		}
		m.sseBackoffs = make(map[string]time.Duration)
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

	m.statusMsg = "Focusing session..."
	m.statusIsError = false
	return m, m.focusTicket(pd.project.Path, r.ticketID)
}

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

	if pd.architect != nil && pd.architect.State == "orphaned" {
		m.showArchitectModeModal = true
		m.architectModeProjectPath = pd.project.Path
		return m, nil
	}

	m.statusMsg = "Spawning architect..."
	m.statusIsError = false
	return m, m.spawnArchitect(pd.project.Path)
}

func (m Model) handleUnlinkArchitect() (tea.Model, tea.Cmd) {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return m, nil
	}
	r := m.rows[m.cursor]

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

func (m Model) handleKillSession() (tea.Model, tea.Cmd) {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return m, nil
	}
	r := m.rows[m.cursor]
	pd := m.projects[r.projectIndex]

	if !pd.project.Exists {
		return m, nil
	}

	if r.kind == rowSession {
		ticket := m.findTicket(pd, r.ticketID)
		if ticket == nil {
			return m, nil
		}
		sessionID := ticket.ID[:8]
		if ticket.IsOrphaned {
			m.killing = true
			m.statusMsg = "Killing orphaned session..."
			m.statusIsError = false
			return m, m.killSession(pd.project.Path, sessionID)
		}
		m.showKillConfirm = true
		m.killProjectPath = pd.project.Path
		m.killSessionID = sessionID
		m.killSessionName = ticket.Title
		return m, nil
	}

	if r.kind == rowProject {
		if pd.architect == nil || (pd.architect.State != "active" && pd.architect.State != "orphaned") {
			return m, nil
		}
		if pd.architect.State == "orphaned" {
			m.killing = true
			m.statusMsg = "Killing orphaned architect..."
			m.statusIsError = false
			return m, m.killSession(pd.project.Path, "architect")
		}
		m.showKillConfirm = true
		m.killProjectPath = pd.project.Path
		m.killSessionID = "architect"
		title := pd.project.Title
		if title == "" {
			title = filepath.Base(pd.project.Path)
		}
		m.killSessionName = title + " architect"
		return m, nil
	}

	return m, nil
}

func (m Model) handleArchitectModeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case isKey(msg, KeyRefresh):
		m.showArchitectModeModal = false
		path := m.architectModeProjectPath
		m.architectModeProjectPath = ""
		m.statusMsg = "Resuming architect..."
		m.statusIsError = false
		return m, m.spawnArchitectWithMode(path, "resume")
	case isKey(msg, KeyFocus):
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
