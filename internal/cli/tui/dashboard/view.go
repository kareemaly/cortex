package dashboard

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/cli/tui/status"
	"github.com/mattn/go-runewidth"
)

func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	if m.showLogViewer {
		return m.logViewer.View()
	}

	// Centered popup overlays — rendered before any content.
	if m.showVariantSelector {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.variantSelector.View())
	}
	if m.showArchitectModeModal {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.renderArchitectModeModal())
	}

	var b strings.Builder

	headerLeft := headerStyle.Render("Cortex Dashboard")
	headerPadding := max(m.width-lipgloss.Width(headerLeft), 0)
	header := headerLeft + strings.Repeat(" ", headerPadding)
	b.WriteString(header)
	b.WriteString("\n\n")

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

	if m.loading {
		b.WriteString(loadingStyle.Render("Loading projects..."))
		return b.String()
	}

	if len(m.projects) == 0 {
		b.WriteString(loadingStyle.Render("No projects registered. Use 'cortex init <name>' to create one."))
		b.WriteString("\n\n")
		b.WriteString(helpBarStyle.Render("[r]efresh  [q]uit"))
		return b.String()
	}

	treeHeight := max(m.height-5, 3)

	m.ensureCursorVisible(treeHeight)

	endIdx := min(m.scrollOffset+treeHeight, len(m.rows))

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
			b.WriteString(m.renderProjectRow(r, selected))
		case rowSession:
			b.WriteString(m.renderSessionRow(r, selected))
		case rowGroup:
			b.WriteString(m.renderGroupRow(r, selected))
		}

		if i < endIdx-1 {
			b.WriteString("\n")
		}
	}

	if endIdx < len(m.rows) {
		b.WriteString("\n")
		b.WriteString(mutedStyleRender.Render("▼"))
	}

	b.WriteString("\n")

	if m.showUnlinkConfirm {
		title := filepath.Base(m.unlinkProjectPath)
		confirmMsg := fmt.Sprintf("Unlink project '%s'? [y]es [n]o", title)
		b.WriteString(warnBadgeStyle.Render(confirmMsg))
		b.WriteString("\n")
		b.WriteString(mutedStyleRender.Render(m.unlinkProjectPath))
		return b.String()
	}

	if m.showKillConfirm {
		name := m.killSessionName
		confirmMsg := fmt.Sprintf("Kill active session '%s'? [y]es [n]o", name)
		b.WriteString(warnBadgeStyle.Render(confirmMsg))
		return b.String()
	}

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

	help := helpBarStyle.Render(helpText())
	badge := m.logBadge()
	if badge != "" {
		help = help + "  " + badge
	}
	b.WriteString(help)

	return b.String()
}

var dashModalBorderStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("214")).
	Padding(1, 2)

var dashModalTitleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("255")).
	MarginBottom(1)

var dashModalHelpStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("240")).
	MarginTop(1)

func (m Model) renderArchitectModeModal() string {
	title := filepath.Base(m.architectModeProjectPath)
	content := dashModalTitleStyle.Render("Orphaned Architect") + "\n" +
		lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render("\""+title+"\"") + "\n" +
		dashModalHelpStyle.Render("[r] resume   [f] fresh   [esc] cancel")
	return dashModalBorderStyle.Render(content)
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

func (m Model) renderProjectRow(r row, selected bool) string {
	pd := m.projects[r.projectIndex]
	indent := ""
	if r.groupName != "" {
		indent = "  "
	}

	architectActive := pd.architect != nil && pd.architect.State == "active"
	architectOrphaned := pd.architect != nil && pd.architect.State == "orphaned"
	indicator := "○"
	if architectOrphaned {
		indicator = "◌"
	} else if architectActive {
		indicator = architectSessionIcon(pd.architect)
	}

	title := pd.project.Title
	if title == "" {
		title = filepath.Base(pd.project.Path)
	}

	archBadge := ""
	if architectOrphaned {
		archBadge = " [arch: orphaned]"
	} else if architectActive && pd.architect.Session != nil {
		archBadge = architectStatusBadge(pd.architect.Session)
	}

	actBadge := activityBadge(pd)

	counts := ""
	if pd.project.Counts != nil {
		c := pd.project.Counts
		counts = fmt.Sprintf("(%d backlog · %d prog · %d done)", c.Backlog, c.Progress, c.Done)
	}

	if pd.loading {
		counts = "(loading...)"
	}

	if pd.err != nil {
		counts = fmt.Sprintf("(error: %s)", pd.err)
	}

	if !pd.project.Exists {
		line := fmt.Sprintf("%s%s %s (stale)", indent, indicator, title)
		if selected {
			return selectedStyle.Render(line)
		}
		return staleStyle.Render(line)
	}

	isActive := pd.isActive()

	if selected {
		plainLine := fmt.Sprintf("%s%s %s%s%s %s", indent, indicator, title, archBadge, actBadge, counts)
		return selectedStyle.Render(plainLine)
	}

	if architectOrphaned {
		return indent + orphanedIconStyle.Render(indicator) + " " + projectStyle.Render(title) + orphanedIconStyle.Render(archBadge) + orphanedIconStyle.Render(actBadge) + " " + countsStyle.Render(counts)
	}
	if architectActive {
		return indent + activeIconStyle.Render(indicator) + " " + projectStyle.Render(title) + activeIconStyle.Render(archBadge) + activeIconStyle.Render(actBadge) + " " + countsStyle.Render(counts)
	}
	if isActive {
		return indent + activeIconStyle.Render(indicator) + " " + projectStyle.Render(title) + activeIconStyle.Render(actBadge) + " " + countsStyle.Render(counts)
	}
	return indent + mutedStyleRender.Render(indicator) + " " + dimmedProjectStyle.Render(title) + " " + countsStyle.Render(counts)
}

func (m Model) renderGroupRow(r row, selected bool) string {
	chevron := "▾"
	suffix := ""
	if m.collapsedGroups[r.groupName] {
		chevron = "▸"
		// Count architects in this group.
		count := 0
		for _, pd := range m.projects {
			if pd.project.Group == r.groupName {
				count++
			}
		}
		suffix = fmt.Sprintf(" (%d)", count)
	}
	line := fmt.Sprintf("%s %s%s", chevron, r.groupName, suffix)
	if selected {
		return selectedStyle.Render(line)
	}
	return groupHeaderStyle.Render(line)
}

func architectSessionIcon(arch *sdk.ArchitectStateResponse) string {
	if arch.Session == nil || arch.Session.Status == nil {
		return status.Icon("")
	}
	return status.Icon(*arch.Session.Status)
}

func architectStatusBadge(sess *sdk.ArchitectSessionResponse) string {
	dur := formatDuration(time.Since(sess.StartedAt))
	if sess.Tool != nil && *sess.Tool != "" {
		return fmt.Sprintf(" [arch: %s %s]", *sess.Tool, dur)
	}
	return fmt.Sprintf(" [arch: %s]", dur)
}

func activityBadge(pd projectData) string {
	if pd.tickets == nil && pd.sessions == nil {
		return ""
	}

	var workerCount, collabCount int
	if pd.tickets != nil {
		for _, t := range pd.tickets.Progress {
			if t.HasActiveSession {
				workerCount++
			}
		}
	}
	if pd.sessions != nil {
		for _, s := range pd.sessions.Sessions {
			if s.SessionType == "collab" && s.Status != "ended" {
				collabCount++
			}
		}
	}

	if workerCount == 0 && collabCount == 0 {
		return ""
	}

	var parts []string
	if workerCount > 0 {
		if workerCount == 1 {
			parts = append(parts, "1 worker")
		} else {
			parts = append(parts, fmt.Sprintf("%d workers", workerCount))
		}
	}
	if collabCount > 0 {
		if collabCount == 1 {
			parts = append(parts, "1 collab")
		} else {
			parts = append(parts, fmt.Sprintf("%d collabs", collabCount))
		}
	}

	return fmt.Sprintf(" [%s]", strings.Join(parts, " · "))
}

func (m Model) renderSessionRow(r row, selected bool) string {
	pd := m.projects[r.projectIndex]
	indent := "    "
	if r.groupName != "" {
		indent = "      "
	}

	if r.sessionType == "collab" {
		session := m.findSession(pd, r.sessionID)
		if session == nil {
			return indent + "???"
		}

		badge := "collab"
		dur := formatDuration(time.Since(session.StartedAt))

		agentLabel := buildAgentLabel(session.Agent, session.Tool)
		agentLabelLen := len(agentLabel)
		if agentLabelLen > 0 {
			agentLabelLen++ // extra space
		}

		nameWidth := m.width - len(indent) - 2 - 1 - agentLabelLen - 1 - len(badge) - 1 - len(dur)
		name := truncateToWidth(session.TicketTitle, nameWidth)

		icon := status.Icon(session.Status)
		styledIcon := activeIconStyle.Render(icon)
		if status.IsEnded(session.Status) {
			styledIcon = status.ApplyEnded(session.Status, icon)
		}
		badgeStyled := progressBadgeStyle.Render(badge)
		if status.IsEnded(session.Status) {
			badgeStyled = status.ApplyEnded(session.Status, badgeStyled)
		}

		if selected {
			plain := fmt.Sprintf("%s%s %s %s %s %s", indent, icon, agentLabel, name, badge, dur)
			return selectedStyle.Render(plain)
		}
		if status.IsEnded(session.Status) {
			return status.ApplyEnded(session.Status,
				fmt.Sprintf("%s%s %s %s %s %s", indent, styledIcon, agentLabel, sessionStyle.Render(name), badgeStyled, durationStyle.Render(dur)))
		}
		if agentLabelLen > 0 {
			return fmt.Sprintf("%s%s %s %s %s %s", indent, styledIcon, agentLabel, sessionStyle.Render(name), badgeStyled, durationStyle.Render(dur))
		}
		return fmt.Sprintf("%s%s %s %s %s", indent, styledIcon, sessionStyle.Render(name), badgeStyled, durationStyle.Render(dur))
	}

	ticket := m.findTicket(pd, r.ticketID)
	if ticket == nil {
		return indent + "???"
	}

	icon := status.TicketIcon(*ticket)
	styledIcon := activeIconStyle.Render(icon)
	if ticket.IsOrphaned {
		styledIcon = orphanedIconStyle.Render(icon)
	}
	name := ticket.Title

	badge := ticket.Status
	if ticket.IsOrphaned {
		badge = "orphaned"
	}
	badgeStyled := progressBadgeStyle.Render(badge)
	if ticket.IsOrphaned {
		badgeStyled = orphanedIconStyle.Render(badge)
	}

	dur := formatDuration(time.Since(ticket.Updated))
	if ticket.SessionStartedAt != nil {
		dur = formatDuration(time.Since(*ticket.SessionStartedAt))
	}

	agentLabel := buildAgentLabel(ticket.Agent, ticket.AgentTool)
	agentLabelLen := len(agentLabel)
	if agentLabelLen > 0 {
		agentLabelLen++ // extra space
	}

	nameWidth := m.width - len(indent) - 2 - 1 - agentLabelLen - 1 - len(badge) - 1 - len(dur)
	name = truncateToWidth(name, nameWidth)

	if selected {
		plain := fmt.Sprintf("%s%s %s %s %s %s", indent, icon, agentLabel, name, badge, dur)
		return selectedStyle.Render(plain)
	}

	if agentLabelLen > 0 {
		return fmt.Sprintf("%s%s %s %s %s %s", indent, styledIcon, agentLabel, sessionStyle.Render(name), badgeStyled, durationStyle.Render(dur))
	}
	return fmt.Sprintf("%s%s %s %s %s", indent, styledIcon, sessionStyle.Render(name), badgeStyled, durationStyle.Render(dur))
}

func truncateToWidth(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if idx := strings.Index(s, "\n"); idx >= 0 {
		s = s[:idx]
	}
	if runewidth.StringWidth(s) <= maxWidth {
		return s
	}
	for i, r := range s {
		if runewidth.StringWidth(s[:i])+runewidth.RuneWidth(r) > maxWidth-1 {
			return s[:i] + "…"
		}
	}
	return s
}

// buildAgentLabel returns "agent_name [Tool]" format, or "agent_name" if no tool,
// or "" if no agent.
func buildAgentLabel(agent string, tool *string) string {
	if agent == "" {
		return ""
	}
	if tool != nil && *tool != "" {
		return fmt.Sprintf("%s [%s]", agent, *tool)
	}
	return agent
}
