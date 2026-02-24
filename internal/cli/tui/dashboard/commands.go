package dashboard

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kareemaly/cortex/internal/cli/sdk"
)

func (m Model) loadProjects() tea.Cmd {
	return func() tea.Msg {
		resp, err := m.globalClient.ListArchitects()
		if err != nil {
			return ArchitectsErrorMsg{Err: err}
		}
		return ArchitectsLoadedMsg{Architects: resp.Architects}
	}
}

func (m Model) loadProjectDetail(projectPath string) tea.Cmd {
	return func() tea.Msg {
		client := sdk.DefaultClient(projectPath)

		tickets, err := client.ListAllTickets("", nil)
		if err != nil {
			return ArchitectDetailLoadedMsg{ArchitectPath: projectPath, Err: err}
		}

		architect, _ := client.GetArchitect()

		return ArchitectDetailLoadedMsg{
			ArchitectPath: projectPath,
			Tickets:       tickets,
			Architect:     architect,
		}
	}
}

func (m Model) subscribeProjectEvents(projectPath string) tea.Cmd {
	return func() tea.Msg {
		client := sdk.DefaultClient(projectPath)
		ctx, cancel := context.WithCancel(context.Background())
		ch, err := client.SubscribeEvents(ctx)
		if err != nil {
			cancel()
			return SSEDisconnectedMsg{ArchitectPath: projectPath}
		}
		return SSEConnectedMsg{ArchitectPath: projectPath, Ch: ch, Cancel: cancel}
	}
}

func (m Model) waitForProjectEvent(projectPath string) tea.Cmd {
	ch, ok := m.sseChannels[projectPath]
	if !ok || ch == nil {
		return nil
	}
	return func() tea.Msg {
		_, ok := <-ch
		if !ok {
			return SSEDisconnectedMsg{ArchitectPath: projectPath}
		}
		return SSEEventMsg{ArchitectPath: projectPath}
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

func (m Model) scheduleSSEReconnect(projectPath string) tea.Cmd {
	backoff := m.sseBackoffs[projectPath]
	return tea.Tick(backoff, func(time.Time) tea.Msg {
		return SSEReconnectTickMsg{ArchitectPath: projectPath}
	})
}

func (m Model) startPollTicker() tea.Cmd {
	return tea.Tick(pollInterval, func(time.Time) tea.Msg {
		return PollTickMsg{}
	})
}

func (m Model) spawnArchitect(projectPath string) tea.Cmd {
	return func() tea.Msg {
		client := sdk.DefaultClient(projectPath)
		_, err := client.SpawnArchitect("")
		return SpawnArchitectMsg{ArchitectPath: projectPath, Err: err}
	}
}

func (m Model) spawnArchitectWithMode(projectPath, mode string) tea.Cmd {
	return func() tea.Msg {
		client := sdk.DefaultClient(projectPath)
		_, err := client.SpawnArchitect(mode)
		return SpawnArchitectMsg{ArchitectPath: projectPath, Err: err}
	}
}

func (m Model) focusTicket(projectPath, ticketID string) tea.Cmd {
	return func() tea.Msg {
		client := sdk.DefaultClient(projectPath)
		if err := client.FocusTicket(ticketID); err != nil {
			return FocusErrorMsg{Err: err}
		}
		return FocusSuccessMsg{Name: ticketID[:8]}
	}
}

func (m Model) unlinkProject(projectPath string) tea.Cmd {
	return func() tea.Msg {
		err := m.globalClient.UnlinkArchitect(projectPath)
		return UnlinkArchitectMsg{ArchitectPath: projectPath, Err: err}
	}
}

func (m Model) killSession(projectPath, sessionID string) tea.Cmd {
	return func() tea.Msg {
		client := sdk.DefaultClient(projectPath)
		err := client.KillSession(sessionID)
		if err != nil {
			return SessionKillErrorMsg{Err: err}
		}
		return SessionKilledMsg{ArchitectPath: projectPath}
	}
}

func (m Model) clearStatusAfterDelay() tea.Cmd {
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return ClearStatusMsg{}
	})
}

func (m Model) tickDuration() tea.Cmd {
	return tea.Tick(30*time.Second, func(time.Time) tea.Msg {
		return TickMsg{}
	})
}
