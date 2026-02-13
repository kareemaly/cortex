package kanban

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/kareemaly/cortex/internal/cli/sdk"
)

// Column represents a kanban column with tickets.
type Column struct {
	title   string
	status  string
	tickets []sdk.TicketSummary
	cursor  int
	vp      viewport.Model
}

// NewColumn creates a new column with the given title and status.
func NewColumn(title, status string) Column {
	return Column{
		title:  title,
		status: status,
	}
}

// Title returns the column title.
func (c *Column) Title() string {
	return c.title
}

// Status returns the column status.
func (c *Column) Status() string {
	return c.status
}

// SelectedTicket returns the currently selected ticket, or nil if empty.
func (c *Column) SelectedTicket() *sdk.TicketSummary {
	if len(c.tickets) == 0 {
		return nil
	}
	return &c.tickets[c.cursor]
}

// MoveUp moves the cursor up within the column.
func (c *Column) MoveUp() {
	if c.cursor > 0 {
		c.cursor--
	}
}

// MoveDown moves the cursor down within the column.
func (c *Column) MoveDown() {
	if c.cursor < len(c.tickets)-1 {
		c.cursor++
	}
}

// JumpToFirst moves cursor to first ticket.
func (c *Column) JumpToFirst() {
	c.cursor = 0
}

// JumpToLast moves cursor to last ticket.
func (c *Column) JumpToLast() {
	if len(c.tickets) > 0 {
		c.cursor = len(c.tickets) - 1
	}
}

// ScrollUp scrolls up by n tickets (for ctrl+u).
func (c *Column) ScrollUp(n int) {
	c.cursor = max(c.cursor-n, 0)
}

// ScrollDown scrolls down by n tickets (for ctrl+d).
func (c *Column) ScrollDown(n int) {
	if len(c.tickets) > 0 {
		c.cursor = min(c.cursor+n, len(c.tickets)-1)
	}
}

// SetTickets sets the tickets for this column and resets cursor if needed.
func (c *Column) SetTickets(tickets []sdk.TicketSummary) {
	c.tickets = tickets
	if c.cursor >= len(tickets) {
		if len(tickets) > 0 {
			c.cursor = len(tickets) - 1
		} else {
			c.cursor = 0
		}
	}
}

// Len returns the number of tickets in the column.
func (c *Column) Len() int {
	return len(c.tickets)
}

// renderAllTickets renders all tickets into a single string for the viewport.
func (c *Column) renderAllTickets(width int, isActive bool) string {
	if len(c.tickets) == 0 {
		emptyText := lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true).
			Render("(empty)")
		return emptyText
	}

	titleWidth := max(width-4, 10)
	var b strings.Builder

	for i, t := range c.tickets {
		isSelected := i == c.cursor && isActive

		// Build type badge
		typeBadge := ""
		if t.Type != "" {
			if isSelected {
				// Use raw ANSI foreground-only change to preserve outer background
				typeBadge = inlineFgColorChange(typeBadgeColorCode(t.Type)) +
					"[" + t.Type + "] " +
					inlineFgColorChange(selectedFgColor)
			} else {
				typeBadge = typeBadgeStyle(t.Type).Render("[" + t.Type + "] ")
			}
		}

		// Build due date indicator
		dueDateIndicator := ""
		if t.Due != nil && c.status == "backlog" {
			now := time.Now()
			if t.Due.Before(now) {
				if isSelected {
					dueDateIndicator = " " + inlineFgColorChange(dueDateColorCode(true)) +
						"[OVERDUE]" +
						inlineFgColorChange(selectedFgColor)
				} else {
					dueDateIndicator = overdueStyle.Render(" [OVERDUE]")
				}
			} else if t.Due.Before(now.Add(24 * time.Hour)) {
				if isSelected {
					dueDateIndicator = " " + inlineFgColorChange(dueDateColorCode(false)) +
						"[DUE SOON]" +
						inlineFgColorChange(selectedFgColor)
				} else {
					dueDateIndicator = dueSoonStyle.Render(" [DUE SOON]")
				}
			}
		}

		// Word wrap title (account for badge width in first line)
		badgeWidth := len(typeBadge)
		if badgeWidth > 0 {
			// Badge uses ANSI escape codes, so calculate actual visible width
			badgeWidth = len("[" + t.Type + "] ")
		}
		wrappedTitle := wrapText(t.Title, titleWidth-badgeWidth)
		if typeBadge != "" && len(wrappedTitle) > 0 {
			wrappedTitle[0] = typeBadge + wrappedTitle[0]
		}
		// Append due date indicator to first line
		if dueDateIndicator != "" && len(wrappedTitle) > 0 {
			wrappedTitle[0] = wrappedTitle[0] + dueDateIndicator
		}

		// Format creation date
		dateStr := t.Created.Format("Jan 2")

		// Build lines based on selection state
		if i == c.cursor && isActive {
			// Selected: show with highlight
			for _, line := range wrappedTitle {
				b.WriteString(selectedTicketStyle.Width(width - 2).Render(line))
				b.WriteString("\n")
			}
			// Metadata line: agent status + date
			meta := ""
			if t.HasActiveSession {
				meta += agentStatusLabel(t) + " · "
			}
			meta += dateStr
			b.WriteString(selectedTicketStyle.Width(width - 2).Render(meta))
		} else {
			// Normal ticket
			for _, line := range wrappedTitle {
				b.WriteString(ticketStyle.Width(width - 2).Render(line))
				b.WriteString("\n")
			}
			// Metadata line: agent status + date
			meta := ""
			if t.HasActiveSession {
				if t.IsOrphaned {
					meta += orphanedStyle.Render(agentStatusLabel(t)) + " · "
				} else {
					meta += activeSessionStyle.Render(agentStatusLabel(t)) + " · "
				}
			}
			meta += dateStr
			b.WriteString(ticketDateStyle.Width(width - 2).Render(meta))
		}

		if i < len(c.tickets)-1 {
			b.WriteString("\n\n")
		}
	}

	return b.String()
}

// cursorYOffset calculates the line number where the cursor's ticket starts in rendered content.
func (c *Column) cursorYOffset(titleWidth int) int {
	y := 0
	for i := 0; i < c.cursor; i++ {
		y += ticketHeight(c.tickets[i], titleWidth)
		y++ // gap line between tickets
	}
	return y
}

// View renders the column.
func (c *Column) View(width int, isActive bool, maxHeight int) string {
	var b strings.Builder

	// Header takes ~2 lines (text + border)
	headerLines := 2

	// Calculate title width for height calculations
	titleWidth := max(width-4, 10)

	totalHeight := totalTicketHeight(c.tickets, titleWidth)
	availableLines := maxHeight - headerLines
	needsScrolling := totalHeight > availableLines

	// Reserve indicator lines when scrolling is needed
	indicatorLines := 0
	if needsScrolling {
		indicatorLines = 2 // top + bottom indicators
	}

	vpHeight := max(availableLines-indicatorLines, 1)

	// Set viewport dimensions
	c.vp.Width = width
	c.vp.Height = vpHeight

	// Render all tickets and set as viewport content
	content := c.renderAllTickets(width, isActive)

	// Preserve scroll position across re-renders
	savedYOffset := c.vp.YOffset
	c.vp.SetContent(content)
	c.vp.SetYOffset(savedYOffset)

	// Ensure cursor is visible within the viewport
	if len(c.tickets) > 0 {
		cursorY := c.cursorYOffset(titleWidth)
		cursorH := ticketHeight(c.tickets[c.cursor], titleWidth)

		// Cursor above viewport — scroll up
		if cursorY < c.vp.YOffset {
			c.vp.SetYOffset(cursorY)
		}
		// Cursor below viewport — scroll down
		if cursorY+cursorH > c.vp.YOffset+vpHeight {
			c.vp.SetYOffset(cursorY + cursorH - vpHeight)
		}
	}

	// Render header with count.
	headerText := fmt.Sprintf("%s (%d)", c.title, len(c.tickets))
	header := columnHeaderStyle(c.status).Width(width - 2).Render(headerText)
	b.WriteString(header)
	b.WriteString("\n")

	// Top scroll indicator
	if needsScrolling && c.vp.YOffset > 0 {
		b.WriteString(mutedStyle.Render("▲"))
		b.WriteString("\n")
	}

	// Viewport content (clipped)
	b.WriteString(c.vp.View())

	// Bottom scroll indicator
	if needsScrolling && c.vp.YOffset+vpHeight < c.vp.TotalLineCount() {
		b.WriteString("\n")
		b.WriteString(mutedStyle.Render("▼"))
	}

	result := b.String()

	// Apply column style with Height for uniform column heights.
	// Safe because viewport already clips content; Height() only adds padding.
	if isActive {
		return activeColumnStyle.Width(width).Height(maxHeight).Render(result)
	}
	return columnStyle.Width(width).Height(maxHeight).Render(result)
}

// agentStatusIcon returns the icon character for the agent's current status.
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

	symbol := symbols[*t.AgentStatus]
	if symbol == "" {
		symbol = "●"
	}
	return symbol
}

// agentStatusLabel returns the icon + truncated tool name (unstyled) for the metadata line.
func agentStatusLabel(t sdk.TicketSummary) string {
	icon := agentStatusIcon(t)
	if t.IsOrphaned {
		return icon + " orphaned"
	}
	if t.AgentTool != nil && *t.AgentTool != "" {
		tool := *t.AgentTool
		if len(tool) > 8 {
			tool = tool[:8]
		}
		return icon + " " + tool
	}
	return icon
}

// wrapText wraps text to fit within width, returning all wrapped lines.
func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	var lines []string
	remaining := text

	for len(remaining) > 0 {
		if len(remaining) <= width {
			lines = append(lines, remaining)
			break
		}

		// Find last space within width for word boundary
		cutPoint := width
		for cutPoint > 0 && remaining[cutPoint] != ' ' {
			cutPoint--
		}
		if cutPoint == 0 {
			cutPoint = width // No space found, hard break
		}

		lines = append(lines, remaining[:cutPoint])
		remaining = strings.TrimLeft(remaining[cutPoint:], " ")
	}

	return lines
}

// ticketHeight returns the number of lines a single ticket occupies (title lines + metadata line).
func ticketHeight(t sdk.TicketSummary, titleWidth int) int {
	bw := 0
	if t.Type != "" {
		bw = len("[" + t.Type + "] ")
	}
	return len(wrapText(t.Title, titleWidth-bw)) + 1
}

// totalTicketHeight returns the total lines needed to render all tickets with gaps.
func totalTicketHeight(tickets []sdk.TicketSummary, titleWidth int) int {
	total := 0
	for i, t := range tickets {
		total += ticketHeight(t, titleWidth)
		if i > 0 {
			total++ // gap line between tickets
		}
	}
	return total
}
