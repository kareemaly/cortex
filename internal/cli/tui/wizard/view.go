package wizard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/kareemaly/cortex/internal/storage"
)

const sidebarWidth = 24

// View renders the full wizard screen.
func (m Model) View() string {
	if !m.ready {
		return "  Initializing..."
	}

	// Layout: sidebar | divider | main
	sidebar := m.renderSidebar()
	main := m.renderMain()

	div := m.renderDivider()

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, div, main)
}

// ---------------------------------------------------------------------------
// Sidebar
// ---------------------------------------------------------------------------

func (m Model) renderSidebar() string {
	w := sidebarWidth
	h := m.height

	var lines []string

	// Header
	header := sidebarHeaderStyle.Width(w).Render(" Setup Wizard")
	lines = append(lines, header)
	lines = append(lines, "")

	for _, def := range allSteps {
		st := m.stepStatus[def.id]
		if st == statusSkipped {
			continue
		}

		var icon, label string
		switch st {
		case statusDone:
			icon = stepDoneStyle.Render("\u2713")
			label = stepDoneStyle.Render(def.label)
		case statusActive:
			icon = stepActiveStyle.Render("\u25b8")
			label = stepActiveStyle.Render(def.label)
		case statusError:
			icon = stepErrorStyle.Render("\u2717")
			label = stepErrorStyle.Render(def.label)
		default:
			icon = stepPendingStyle.Render("\u2022")
			label = stepPendingStyle.Render(def.label)
		}

		lines = append(lines, fmt.Sprintf("  %s %s", icon, label))
	}

	content := strings.Join(lines, "\n")

	return lipgloss.NewStyle().
		Width(w).
		Height(h).
		Render(content)
}

// ---------------------------------------------------------------------------
// Divider
// ---------------------------------------------------------------------------

func (m Model) renderDivider() string {
	div := strings.Repeat("\u2502", 1)
	var lines []string
	for i := 0; i < m.height; i++ {
		lines = append(lines, dividerStyle.Render(div))
	}
	return strings.Join(lines, "\n")
}

// ---------------------------------------------------------------------------
// Main pane
// ---------------------------------------------------------------------------

func (m Model) renderMain() string {
	w := m.width - sidebarWidth - 1 // 1 for divider
	if w < 20 {
		w = 20
	}
	h := m.height

	var content string

	switch m.step {
	case stepDetectAgents:
		content = m.viewDetecting(w)
	case stepSelectAgent:
		content = m.viewSelectAgent(w)
	case stepInputName:
		content = m.viewInputName(w)
	case stepSelectModel:
		content = m.viewSelectModel(w)
	case stepInputRepos:
		content = m.viewInputRepos(w)
	case stepCompanion:
		content = m.viewConfirm(w, "Use "+m.companion+" companion?")
	case stepFeatureBranches:
		content = m.viewConfirm(w, "Feature branches per ticket?")
	case stepResearchPaths:
		content = m.viewResearchPaths(w)
	case stepInstall:
		content = m.viewInstalling(w)
	case stepAgentsMDConfirm:
		outputFile := "AGENTS.md"
		if m.agent == "claude" {
			outputFile = "CLAUDE.md"
		}
		content = m.viewConfirm(w, "Generate "+outputFile+" from repos?")
	case stepAgentsMDRun:
		content = m.viewAgentStream(w)
	case stepDaemon:
		content = m.viewDaemon(w)
	case stepDone:
		content = m.viewDone(w, h)
	}

	return lipgloss.NewStyle().
		Width(w).
		Height(h).
		PaddingLeft(2).
		Render(content)
}

// ---------------------------------------------------------------------------
// Step views
// ---------------------------------------------------------------------------

func (m Model) viewDetecting(w int) string {
	var b strings.Builder
	b.WriteString(mainHeaderStyle.Width(w - 2).Render(" Detect Agents"))
	b.WriteString("\n\n")
	b.WriteString("  " + spinnerStyle.Render(m.spinner.View()) + " Scanning PATH for coding agents...")
	return b.String()
}

func (m Model) viewSelectAgent(w int) string {
	var b strings.Builder
	b.WriteString(mainHeaderStyle.Width(w - 2).Render(" Select Agent"))
	b.WriteString("\n\n")

	// Show what was detected
	if m.agents.ClaudeAvailable {
		b.WriteString("  " + resultCheckStyle.Render("\u2713") + " claude found\n")
	}
	if m.agents.OpenCodeAvailable {
		b.WriteString("  " + resultCheckStyle.Render("\u2713") + " opencode found\n")
	}
	b.WriteString("\n")

	b.WriteString("  " + promptLabelStyle.Render("Which agent?") + "\n\n")

	for i, opt := range m.selectOpts {
		if i == m.selectIdx {
			b.WriteString("  " + selectedItemStyle.Render(" \u25b8 "+opt+" ") + "\n")
		} else {
			b.WriteString("    " + normalItemStyle.Render(opt) + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString("  " + hintStyle.Render("j/k or \u2191/\u2193 to navigate, Enter to select"))
	return b.String()
}

func (m Model) viewInputName(w int) string {
	var b strings.Builder
	b.WriteString(mainHeaderStyle.Width(w - 2).Render(" Project Name"))
	b.WriteString("\n\n")

	b.WriteString("  " + promptLabelStyle.Render("Name:") + " " + m.textInput.View() + "\n\n")
	b.WriteString("  " + hintStyle.Render("This will create a workspace directory at"))
	b.WriteString("\n")
	val := m.textInput.Value()
	if val == "" {
		b.WriteString("  " + hintStyle.Render(m.cfg.Cwd+"/<name>/"))
	} else {
		slug := storage.GenerateSlug(val, val)
		b.WriteString("  " + hintStyle.Render(filepath.Join(m.cfg.Cwd, slug)+"/"))
	}
	b.WriteString("\n\n")
	b.WriteString("  " + hintStyle.Render("Enter to confirm"))
	return b.String()
}

func (m Model) viewSelectModel(w int) string {
	var b strings.Builder
	b.WriteString(mainHeaderStyle.Width(w - 2).Render(" Select Model"))
	b.WriteString("\n\n")

	if len(m.models) == 0 {
		b.WriteString("  " + spinnerStyle.Render(m.spinner.View()) + " Loading models from opencode...")
		return b.String()
	}

	b.WriteString("  " + promptLabelStyle.Render("Model:") + " " + m.textInput.View() + "\n\n")

	filtered := m.getFilteredOpts()
	maxShow := m.height - 10
	if maxShow < 5 {
		maxShow = 5
	}
	if maxShow > 20 {
		maxShow = 20
	}

	start, end := scrollWindow(m.selectIdx, len(filtered), maxShow)

	for i := start; i < end; i++ {
		opt := filtered[i]
		if i == m.selectIdx {
			b.WriteString("  " + selectedItemStyle.Render(" \u25b8 "+opt+" ") + "\n")
		} else {
			b.WriteString("    " + normalItemStyle.Render(opt) + "\n")
		}
	}

	if len(filtered) > maxShow {
		b.WriteString("\n")
		b.WriteString("  " + hintStyle.Render(fmt.Sprintf("showing %d\u2013%d of %d", start+1, end, len(filtered))))
	}

	b.WriteString("\n\n")
	b.WriteString("  " + hintStyle.Render("Type to filter, \u2191/\u2193 to navigate, Enter to select"))
	return b.String()
}

func (m Model) viewInputRepos(w int) string {
	var b strings.Builder
	b.WriteString(mainHeaderStyle.Width(w - 2).Render(" Repositories"))
	b.WriteString("\n\n")

	b.WriteString("  " + promptLabelStyle.Render("Repos (comma-separated):") + "\n\n")
	b.WriteString("  " + m.textInput.View() + "\n")

	if m.repoError != "" {
		b.WriteString("\n  " + errorStatusStyle.Render(m.repoError))
	}

	b.WriteString("\n\n")
	b.WriteString("  " + hintStyle.Render("Paths to git repositories the architect manages."))
	b.WriteString("\n")
	b.WriteString("  " + hintStyle.Render("Example: ~/projects/api, ~/projects/frontend"))
	b.WriteString("\n\n")
	b.WriteString("  " + hintStyle.Render("Enter to confirm"))
	return b.String()
}

func (m Model) viewConfirm(w int, prompt string) string {
	var b strings.Builder
	// Pick a header from the step label
	header := prompt
	if len(header) > w-4 {
		header = header[:w-4]
	}
	b.WriteString(mainHeaderStyle.Width(w - 2).Render(" " + header))
	b.WriteString("\n\n")

	b.WriteString("  " + promptLabelStyle.Render(prompt) + " ")

	if m.confirmDflt {
		b.WriteString(confirmYesStyle.Render("[Y]") + "/" + confirmNoStyle.Render("n"))
	} else {
		b.WriteString(confirmYesStyle.Render("y") + "/" + confirmNoStyle.Render("[N]"))
	}

	b.WriteString("\n\n")
	b.WriteString("  " + hintStyle.Render("y/n or Enter for default"))
	return b.String()
}

func (m Model) viewResearchPaths(w int) string {
	var b strings.Builder
	b.WriteString(mainHeaderStyle.Width(w - 2).Render(" Research Paths"))
	b.WriteString("\n\n")

	b.WriteString("  " + promptLabelStyle.Render("Research paths:") + "\n\n")
	b.WriteString("  " + m.textInput.View() + "\n\n")
	b.WriteString("  " + hintStyle.Render("Comma-separated paths for the research agent."))
	b.WriteString("\n")
	b.WriteString("  " + hintStyle.Render("Default: ~/**"))
	b.WriteString("\n\n")
	b.WriteString("  " + hintStyle.Render("Enter to confirm"))
	return b.String()
}

func (m Model) viewInstalling(w int) string {
	var b strings.Builder
	b.WriteString(mainHeaderStyle.Width(w - 2).Render(" Installing"))
	b.WriteString("\n\n")
	b.WriteString("  " + spinnerStyle.Render(m.spinner.View()) + " Setting up workspace...")
	return b.String()
}

func (m Model) viewAgentStream(w int) string {
	var b strings.Builder
	outputFile := "AGENTS.md"
	if m.agent == "claude" {
		outputFile = "CLAUDE.md"
	}
	b.WriteString(mainHeaderStyle.Width(w - 2).Render(" Generating " + outputFile))
	b.WriteString("\n\n")

	b.WriteString("  " + spinnerStyle.Render(m.spinner.View()) + " ")
	b.WriteString(streamTitleStyle.Render("Analyzing repositories..."))

	if m.agentTotalCost > 0 || m.agentTotalTok.input > 0 {
		b.WriteString(" " + streamStatsStyle.Render(formatStats(m.agentTotalCost, m.agentTotalTok)))
	}
	b.WriteString("\n\n")

	m.agentStreamMu.Lock()
	for _, msg := range m.agentStreamMsgs {
		switch msg.msgType {
		case "tool":
			icon := toolIcon(msg.tool)
			switch msg.status {
			case "completed":
				b.WriteString("  " + streamDoneStyle.Render(icon+" "+msg.tool))
			case "error":
				b.WriteString("  " + streamErrStyle.Render("\u2717 "+msg.tool))
			default:
				b.WriteString("  " + streamToolStyle.Render(icon+" "+msg.tool))
			}
			if msg.content != "" {
				b.WriteString(" " + streamContentStyle.Render(msg.content))
			}
		case "subagent":
			if msg.subagent != nil {
				switch msg.status {
				case "completed":
					b.WriteString("  " + streamDoneStyle.Render("\u25c8 "+msg.subagent.agentType))
				case "error":
					b.WriteString("  " + streamErrStyle.Render("\u2717 "+msg.subagent.agentType))
				default:
					b.WriteString("  " + streamSubagentStyle.Render("\u25c8 "+msg.subagent.agentType))
				}
				b.WriteString(" " + streamContentStyle.Render(msg.subagent.description))
			}
		case "text":
			b.WriteString("  " + streamContentStyle.Render(strings.TrimSpace(strings.ReplaceAll(msg.content, "\n", " "))))
		case "reasoning":
			b.WriteString("  " + streamReasonStyle.Render("\U0001f4ad "+strings.TrimSpace(strings.ReplaceAll(msg.content, "\n", " "))))
		}
		b.WriteString("\n")
	}
	m.agentStreamMu.Unlock()

	return b.String()
}

func (m Model) viewDaemon(w int) string {
	var b strings.Builder
	b.WriteString(mainHeaderStyle.Width(w - 2).Render(" Starting Daemon"))
	b.WriteString("\n\n")
	b.WriteString("  " + spinnerStyle.Render(m.spinner.View()) + " Starting cortexd...")
	return b.String()
}

func (m Model) viewDone(w, h int) string {
	var b strings.Builder
	b.WriteString(mainHeaderStyle.Width(w - 2).Render(" Complete"))
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString("  " + errorStatusStyle.Render("Error: "+m.err.Error()) + "\n")
		b.WriteString("\n")
		b.WriteString("  " + hintStyle.Render("Press any key to exit"))
		return b.String()
	}

	// Summary of collected values
	if m.agent != "" {
		b.WriteString("  " + resultHeaderStyle.Render("Agent:") + " " + m.agent + "\n")
	}
	if m.name != "" {
		b.WriteString("  " + resultHeaderStyle.Render("Name:") + "  " + m.name + "\n")
	}
	if m.model != "" {
		b.WriteString("  " + resultHeaderStyle.Render("Model:") + " " + m.model + "\n")
	}
	if len(m.repos) > 0 {
		b.WriteString("  " + resultHeaderStyle.Render("Repos:") + " " + strings.Join(m.repos, ", ") + "\n")
	}
	b.WriteString("\n")

	// Install result
	if m.installResult != nil {
		m.renderInstallResult(&b)
		b.WriteString("\n")
	}

	// AGENTS.md
	if m.generateMD {
		outputFile := "AGENTS.md"
		if m.agent == "claude" {
			outputFile = "CLAUDE.md"
		}
		switch m.stepStatus[stepAgentsMDRun] {
		case statusDone:
			b.WriteString("  " + resultCheckStyle.Render("\u2713") + " Generated " + filepath.Join(m.architectPath, outputFile) + "\n")
		case statusError:
			b.WriteString("  " + resultCrossStyle.Render("\u2717") + " Failed to generate " + outputFile + "\n")
		}
	}

	// Daemon
	if m.daemonErr != nil {
		b.WriteString("  " + resultCrossStyle.Render("\u2717") + " Failed to start daemon: " + m.daemonErr.Error() + "\n")
		b.WriteString("    Run 'cortex daemon restart' to try again\n")
	} else {
		b.WriteString("  " + resultCheckStyle.Render("\u2713") + " Daemon running\n")
	}

	b.WriteString("\n")
	if m.slug != "" {
		b.WriteString("  Run " + promptLabelStyle.Render("cortex architect start "+m.slug) + " to begin.\n")
	}
	b.WriteString("\n")
	b.WriteString("  " + hintStyle.Render("Press any key to exit"))

	return b.String()
}

func (m Model) renderInstallResult(b *strings.Builder) {
	r := m.installResult
	homeDir, _ := os.UserHomeDir()
	shorten := func(path string) string {
		if homeDir != "" && strings.HasPrefix(path, homeDir) {
			return "~" + path[len(homeDir):]
		}
		return path
	}

	b.WriteString("  " + resultHeaderStyle.Render("Global:") + "\n")
	for _, item := range r.GlobalItems {
		p := shorten(item.Path)
		switch item.Status {
		case statusCreated:
			b.WriteString("    " + resultCheckStyle.Render("\u2713") + " Created " + p + "\n")
		case statusExists:
			b.WriteString("    " + resultBulletStyle.Render("\u2022") + " " + p + "\n")
		}
	}

	if len(r.ArchitectItems) > 0 {
		b.WriteString("  " + resultHeaderStyle.Render(fmt.Sprintf("Architect (%s):", r.ArchitectName)) + "\n")
		for _, item := range r.ArchitectItems {
			p := shorten(item.Path)
			switch item.Status {
			case statusCreated:
				b.WriteString("    " + resultCheckStyle.Render("\u2713") + " Created " + p + "\n")
			case statusExists:
				b.WriteString("    " + resultBulletStyle.Render("\u2022") + " " + p + "\n")
			}
		}

		if r.RegistrationError != nil {
			b.WriteString("    " + resultCrossStyle.Render("\u2717") + " Registration failed: " + r.RegistrationError.Error() + "\n")
		} else if r.Registered {
			b.WriteString("    " + resultCheckStyle.Render("\u2713") + " Registered in ~/.cortex/settings.yaml\n")
		} else {
			b.WriteString("    " + resultBulletStyle.Render("\u2022") + " Already registered\n")
		}
	}

	b.WriteString("  " + resultHeaderStyle.Render("Dependencies:") + "\n")
	for _, dep := range r.Dependencies {
		if dep.Available {
			b.WriteString("    " + resultCheckStyle.Render("\u2713") + " " + dep.Name + "\n")
		} else {
			b.WriteString("    " + resultCrossStyle.Render("\u2717") + " " + dep.Name + " not found\n")
		}
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func scrollWindow(selected, total, maxShow int) (int, int) {
	start := 0
	if total > maxShow && selected >= maxShow-2 {
		start = selected - (maxShow - 3)
		if start+maxShow > total {
			start = total - maxShow
		}
	}
	end := start + maxShow
	if end > total {
		end = total
	}
	return start, end
}

func formatStats(cost float64, tokens agentStreamTokens) string {
	var parts []string
	if tokens.input > 0 {
		parts = append(parts, fmt.Sprintf("%dk in", tokens.input/1000))
	}
	if tokens.output > 0 {
		parts = append(parts, fmt.Sprintf("%dk out", tokens.output/1000))
	}
	if cost > 0 {
		parts = append(parts, fmt.Sprintf("$%.4f", cost))
	}
	if len(parts) == 0 {
		return ""
	}
	return "[" + strings.Join(parts, " | ") + "]"
}

func toolIcon(tool string) string {
	switch tool {
	case "bash":
		return "\u25b6"
	case "read":
		return "\U0001f4c4"
	case "write", "edit":
		return "\u270e"
	case "glob", "grep":
		return "\U0001f50d"
	case "task":
		return "\u25c8"
	default:
		return "\u2022"
	}
}

// statusCreated and statusExists map install.ItemStatus values for comparison
// without importing the install package's type system in the view layer.
const (
	statusCreated = 0 // install.StatusCreated
	statusExists  = 1 // install.StatusExists
)
