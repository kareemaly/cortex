package dashboard

import (
	"fmt"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kareemaly/cortex/internal/cli/tui/tuilog"
)

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

	case ArchitectsLoadedMsg:
		m.loading = false
		m.err = nil
		m.projects = make([]projectData, len(msg.Architects))
		var cmds []tea.Cmd
		for i, p := range msg.Architects {
			m.projects[i] = projectData{project: p, loading: true}
			if p.Exists {
				cmds = append(cmds, m.loadProjectDetail(p.Path))
				cmds = append(cmds, m.subscribeProjectEvents(p.Path))
			}
		}
		m.rebuildRows()
		m.logBuf.Debugf("api", "projects loaded: %d", len(msg.Architects))
		return m, tea.Batch(cmds...)

	case ArchitectsErrorMsg:
		m.loading = false
		m.err = msg.Err
		m.logBuf.Errorf("api", "failed to load projects: %s", msg.Err)
		return m, nil

	case ArchitectDetailLoadedMsg:
		idx := m.findProject(msg.ArchitectPath)
		if idx < 0 {
			return m, nil
		}
		m.projects[idx].loading = false
		if msg.Err != nil {
			m.projects[idx].err = msg.Err
			m.logBuf.Errorf("api", "failed to load project detail: %s: %s", filepath.Base(msg.ArchitectPath), msg.Err)
		} else {
			m.projects[idx].tickets = msg.Tickets
			m.projects[idx].architect = msg.Architect
			m.projects[idx].err = nil
			m.logBuf.Debugf("api", "project detail loaded: %s", filepath.Base(msg.ArchitectPath))
		}
		m.rebuildRows()
		return m, nil

	case SSEConnectedMsg:
		if oldCancel, ok := m.sseContexts[msg.ArchitectPath]; ok {
			oldCancel()
		}
		m.sseContexts[msg.ArchitectPath] = msg.Cancel
		m.sseChannels[msg.ArchitectPath] = msg.Ch
		delete(m.sseBackoffs, msg.ArchitectPath)
		m.logBuf.Infof("sse", "connected: %s", filepath.Base(msg.ArchitectPath))
		idx := m.findProject(msg.ArchitectPath)
		cmds := []tea.Cmd{m.waitForProjectEvent(msg.ArchitectPath)}
		if idx >= 0 {
			cmds = append(cmds, m.loadProjectDetail(msg.ArchitectPath))
		}
		return m, tea.Batch(cmds...)

	case SSEEventMsg:
		m.logBuf.Debugf("sse", "event: %s", filepath.Base(msg.ArchitectPath))
		idx := m.findProject(msg.ArchitectPath)
		cmds := []tea.Cmd{m.waitForProjectEvent(msg.ArchitectPath)}
		if idx >= 0 {
			cmds = append(cmds, m.loadProjectDetail(msg.ArchitectPath))
		}
		return m, tea.Batch(cmds...)

	case SSEDisconnectedMsg:
		if ch, ok := m.sseChannels[msg.ArchitectPath]; ok && ch != nil {
			return m, nil
		}
		if cancel, ok := m.sseContexts[msg.ArchitectPath]; ok {
			cancel()
			delete(m.sseContexts, msg.ArchitectPath)
		}
		delete(m.sseChannels, msg.ArchitectPath)
		m.sseBackoffs[msg.ArchitectPath] = nextBackoff(m.sseBackoffs[msg.ArchitectPath])
		m.logBuf.Warnf("sse", "disconnected: %s, reconnecting in %s", filepath.Base(msg.ArchitectPath), m.sseBackoffs[msg.ArchitectPath])
		return m, m.scheduleSSEReconnect(msg.ArchitectPath)

	case SSEReconnectTickMsg:
		if m.findProject(msg.ArchitectPath) >= 0 {
			m.logBuf.Debugf("sse", "attempting reconnect: %s", filepath.Base(msg.ArchitectPath))
			return m, m.subscribeProjectEvents(msg.ArchitectPath)
		}
		return m, nil

	case PollTickMsg:
		var cmds []tea.Cmd
		for _, pd := range m.projects {
			if pd.project.Exists {
				cmds = append(cmds, m.loadProjectDetail(pd.project.Path))
			}
		}
		cmds = append(cmds, m.startPollTicker())
		return m, tea.Batch(cmds...)

	case SpawnArchitectMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Spawn error: %s", msg.Err)
			m.statusIsError = true
			m.logBuf.Errorf("spawn", "architect spawn failed: %s: %s", filepath.Base(msg.ArchitectPath), msg.Err)
		} else {
			m.statusMsg = "Architect spawned"
			m.statusIsError = false
			m.logBuf.Infof("spawn", "architect spawned: %s", filepath.Base(msg.ArchitectPath))
			idx := m.findProject(msg.ArchitectPath)
			if idx >= 0 {
				return m, tea.Batch(m.loadProjectDetail(msg.ArchitectPath), m.clearStatusAfterDelay())
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

	case UnlinkArchitectMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Unlink error: %s", msg.Err)
			m.statusIsError = true
			m.logBuf.Errorf("unlink", "unlink failed: %s: %s", filepath.Base(msg.ArchitectPath), msg.Err)
		} else {
			m.statusMsg = "Project unlinked"
			m.statusIsError = false
			m.logBuf.Infof("unlink", "project unlinked: %s", filepath.Base(msg.ArchitectPath))
			if cancel, ok := m.sseContexts[msg.ArchitectPath]; ok {
				cancel()
				delete(m.sseContexts, msg.ArchitectPath)
				delete(m.sseChannels, msg.ArchitectPath)
			}
			m.loading = true
			return m, tea.Batch(m.loadProjects(), m.clearStatusAfterDelay())
		}
		return m, m.clearStatusAfterDelay()

	case SessionKilledMsg:
		m.killing = false
		m.showKillConfirm = false
		m.statusMsg = "Session killed"
		m.statusIsError = false
		m.logBuf.Infof("kill", "session killed: %s", filepath.Base(msg.ArchitectPath))
		idx := m.findProject(msg.ArchitectPath)
		if idx >= 0 {
			return m, tea.Batch(m.loadProjectDetail(msg.ArchitectPath), m.clearStatusAfterDelay())
		}
		return m, m.clearStatusAfterDelay()

	case SessionKillErrorMsg:
		m.killing = false
		m.showKillConfirm = false
		m.statusMsg = fmt.Sprintf("Kill error: %s", msg.Err)
		m.statusIsError = true
		m.logBuf.Errorf("kill", "session kill failed: %s", msg.Err)
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
