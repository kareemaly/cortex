package dashboard

import (
	"fmt"
	"slices"
	"time"

	"github.com/kareemaly/cortex/internal/cli/sdk"
)

func (m *Model) rebuildRows() {
	sortProjects := func(projects []projectData) {
		slices.SortStableFunc(projects, func(a, b projectData) int {
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
	}

	appendProjectRows := func(rows []row, projectIndices []int, groupName string) []row {
		for _, i := range projectIndices {
			pd := m.projects[i]
			rows = append(rows, row{kind: rowProject, projectIndex: i, groupName: groupName})

			if pd.tickets != nil {
				var sessionTickets []sdk.TicketSummary
				for _, t := range pd.tickets.Progress {
					if t.HasActiveSession {
						sessionTickets = append(sessionTickets, t)
					}
				}
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
					rows = append(rows, row{kind: rowSession, projectIndex: i, ticketID: t.ID, sessionType: "ticket", groupName: groupName})
				}
			}

			if pd.sessions != nil {
				var collabSessions []sdk.SessionListItem
				for _, s := range pd.sessions.Sessions {
					if s.SessionType == "collab" {
						collabSessions = append(collabSessions, s)
					}
				}
				slices.SortStableFunc(collabSessions, func(a, b sdk.SessionListItem) int {
					if a.StartedAt.After(b.StartedAt) {
						return -1
					}
					if b.StartedAt.After(a.StartedAt) {
						return 1
					}
					return 0
				})
				for _, s := range collabSessions {
					rows = append(rows, row{kind: rowSession, projectIndex: i, sessionType: "collab", sessionID: s.SessionID, groupName: groupName})
				}
			}
		}
		return rows
	}

	// Bucket projects by group. "" means ungrouped.
	groupMap := make(map[string][]int)
	for i, pd := range m.projects {
		g := pd.project.Group
		groupMap[g] = append(groupMap[g], i)
	}

	// Collect named group names sorted alphabetically.
	var groupNames []string
	for g := range groupMap {
		if g != "" {
			groupNames = append(groupNames, g)
		}
	}
	slices.Sort(groupNames)

	// Sort each bucket internally.
	for g, indices := range groupMap {
		subset := make([]projectData, len(indices))
		for j, idx := range indices {
			subset[j] = m.projects[idx]
		}
		sortProjects(subset)
		for j, pd := range subset {
			// Find original index by path since we sorted a copy.
			for k, p := range m.projects {
				if p.project.Path == pd.project.Path {
					groupMap[g][j] = k
					break
				}
			}
		}
	}

	var rows []row

	// Named groups first (sorted).
	for _, g := range groupNames {
		indices := groupMap[g]
		rows = append(rows, row{kind: rowGroup, groupName: g})
		if !m.collapsedGroups[g] {
			rows = appendProjectRows(rows, indices, g)
		}
	}

	// Ungrouped architects last, no header.
	if ungrouped, ok := groupMap[""]; ok {
		rows = appendProjectRows(rows, ungrouped, "")
	}

	m.rows = rows

	if len(m.rows) > 0 {
		if m.cursor >= len(m.rows) {
			m.cursor = len(m.rows) - 1
		}
	} else {
		m.cursor = 0
	}
}

func (m Model) findProject(path string) int {
	for i, pd := range m.projects {
		if pd.project.Path == path {
			return i
		}
	}
	return -1
}

func (m Model) findTicket(pd projectData, ticketID string) *sdk.TicketSummary {
	if pd.tickets == nil {
		return nil
	}
	for i := range pd.tickets.Progress {
		if pd.tickets.Progress[i].ID == ticketID {
			return &pd.tickets.Progress[i]
		}
	}
	return nil
}

func (m Model) findSession(pd projectData, sessionID string) *sdk.SessionListItem {
	if pd.sessions == nil {
		return nil
	}
	for i := range pd.sessions.Sessions {
		if pd.sessions.Sessions[i].SessionID == sessionID {
			return &pd.sessions.Sessions[i]
		}
	}
	return nil
}

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
	}
	if pd.sessions != nil {
		for _, s := range pd.sessions.Sessions {
			if s.StartedAt.After(newest) {
				newest = s.StartedAt
			}
		}
	}
	return newest
}

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

func agentStatusIcon(t sdk.TicketSummary) string {
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
