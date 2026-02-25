package dashboard

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/kareemaly/cortex/internal/cli/sdk"
)

func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	if m.showLogViewer {
		return m.logViewer.View()
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
		b.WriteString(loadingStyle.Render("No projects registered. Use 'cortex architect create' in a project directory."))
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
			b.WriteString(m.renderProjectRow(r.projectIndex, selected))
		case rowSession:
			b.WriteString(m.renderSessionRow(r, selected))
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

	if m.showArchitectModeModal {
		title := filepath.Base(m.architectModeProjectPath)
		prompt := fmt.Sprintf("Orphaned architect for '%s'", title)
		options := "[r]esume  [f]resh  [esc] cancel"
		b.WriteString(warnBadgeStyle.Render(prompt))
		b.WriteString("\n")
		b.WriteString(helpBarStyle.Render(options))
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

func (m Model) renderProjectRow(projectIdx int, selected bool) string {
	pd := m.projects[projectIdx]

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

func architectStatusBadge(sess *sdk.ArchitectSessionResponse) string {
	dur := formatDuration(time.Since(sess.StartedAt))
	if sess.Tool != nil && *sess.Tool != "" {
		return fmt.Sprintf(" [arch: %s %s]", *sess.Tool, dur)
	}
	return fmt.Sprintf(" [arch: %s]", dur)
}

func (m Model) renderSessionRow(r row, selected bool) string {
	pd := m.projects[r.projectIndex]
	indent := "    "

	if r.sessionType == "collab" {
		session := m.findSession(pd, r.sessionID)
		if session == nil {
			return indent + "???"
		}

		name := session.TicketTitle

		icon := "●"
		styledIcon := activeIconStyle.Render(icon)

		badge := "collab"
		badgeStyled := progressBadgeStyle.Render(badge)

		dur := formatDuration(time.Since(session.StartedAt))

		if selected {
			plain := fmt.Sprintf("%s%s %s %s %s", indent, icon, name, badge, dur)
			return selectedStyle.Render(plain)
		}

		return fmt.Sprintf("%s%s %s %s %s", indent, styledIcon, sessionStyle.Render(name), badgeStyled, durationStyle.Render(dur))
	}

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

	if selected {
		plain := fmt.Sprintf("%s%s %s %s %s", indent, icon, name, badge, dur)
		return selectedStyle.Render(plain)
	}

	return fmt.Sprintf("%s%s %s %s %s", indent, styledIcon, sessionStyle.Render(name), badgeStyled, durationStyle.Render(dur))
}
