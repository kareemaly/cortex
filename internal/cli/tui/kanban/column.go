package kanban

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/kareemaly/cortex/internal/cli/sdk"
)

const linesPerTicket = 6 // title (up to 5 lines) + date line

// Column represents a kanban column with tickets.
type Column struct {
	title        string
	status       string
	tickets      []sdk.TicketSummary
	cursor       int
	scrollOffset int // first visible ticket index
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
	c.scrollOffset = 0
}

// JumpToLast moves cursor to last ticket.
func (c *Column) JumpToLast() {
	if len(c.tickets) > 0 {
		c.cursor = len(c.tickets) - 1
	}
	// scrollOffset adjusted in EnsureCursorVisible
}

// ScrollUp scrolls up by n tickets (for ctrl+u).
func (c *Column) ScrollUp(n int) {
	c.cursor = max(c.cursor-n, 0)
	// scrollOffset adjusted in EnsureCursorVisible
}

// ScrollDown scrolls down by n tickets (for ctrl+d).
func (c *Column) ScrollDown(n int) {
	if len(c.tickets) > 0 {
		c.cursor = min(c.cursor+n, len(c.tickets)-1)
	}
	// scrollOffset adjusted in EnsureCursorVisible
}

// EnsureCursorVisible adjusts scrollOffset to keep cursor in view.
func (c *Column) EnsureCursorVisible(visibleCount int) {
	if visibleCount <= 0 {
		return
	}
	// Cursor above visible area
	if c.cursor < c.scrollOffset {
		c.scrollOffset = c.cursor
	}
	// Cursor below visible area
	if c.cursor >= c.scrollOffset+visibleCount {
		c.scrollOffset = c.cursor - visibleCount + 1
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
	// Reset scroll offset if it's now invalid
	if c.scrollOffset >= len(tickets) {
		c.scrollOffset = max(len(tickets)-1, 0)
	}
}

// Len returns the number of tickets in the column.
func (c *Column) Len() int {
	return len(c.tickets)
}

// View renders the column.
func (c *Column) View(width int, isActive bool, maxHeight int) string {
	var b strings.Builder

	// Header takes ~2 lines (text + border)
	headerLines := 2
	// Scroll indicators take 1 line each when shown
	indicatorLines := 2 // Reserve space for both
	visibleCount := max((maxHeight-headerLines-indicatorLines)/linesPerTicket, 1)

	// Ensure cursor is visible
	c.EnsureCursorVisible(visibleCount)

	// Render header with count.
	headerText := fmt.Sprintf("%s (%d)", c.title, len(c.tickets))
	header := columnHeaderStyle(c.status).Width(width - 2).Render(headerText)
	b.WriteString(header)
	b.WriteString("\n")

	// Top scroll indicator
	if c.scrollOffset > 0 {
		b.WriteString(mutedStyle.Render("  ▲"))
		b.WriteString("\n")
	} else {
		b.WriteString("\n") // Empty line to maintain layout
	}

	// Render only visible tickets.
	if len(c.tickets) == 0 {
		emptyText := lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true).
			Render("(empty)")
		b.WriteString(emptyText)
		b.WriteString("\n")
	} else {
		endIdx := min(c.scrollOffset+visibleCount, len(c.tickets))
		for i := c.scrollOffset; i < endIdx; i++ {
			t := c.tickets[i]

			// Calculate available width for title text
			titleWidth := max(width-6, 10)

			// Word wrap title
			wrappedTitle := wrapText(t.Title, titleWidth)

			// Format creation date
			dateStr := t.Created.Format("Jan 2")

			// Build lines based on selection state
			if i == c.cursor && isActive {
				// Selected: show with prefix and highlight
				for lineIdx, line := range wrappedTitle {
					prefix := "  "
					if lineIdx == 0 {
						prefix = "> "
					}
					b.WriteString(selectedTicketStyle.Width(width - 2).Render(prefix + line))
					b.WriteString("\n")
				}
				// Date line with selection style
				b.WriteString(selectedTicketStyle.Width(width - 2).Render("  " + dateStr))
			} else {
				// Normal: show with appropriate prefix
				for lineIdx, line := range wrappedTitle {
					prefix := "  "
					if lineIdx == 0 && t.HasActiveSession {
						prefix = formatAgentStatus(t)
					}
					b.WriteString(ticketStyle.Width(width - 2).Render(prefix + line))
					b.WriteString("\n")
				}
				// Date line with muted style
				b.WriteString(ticketDateStyle.Width(width - 2).Render("  " + dateStr))
			}

			if i < endIdx-1 {
				b.WriteString("\n")
			}
		}
	}

	// Bottom scroll indicator
	b.WriteString("\n")
	if c.scrollOffset+visibleCount < len(c.tickets) {
		b.WriteString(mutedStyle.Render("  ▼"))
	}

	content := b.String()

	// Apply column style.
	if isActive {
		return activeColumnStyle.Width(width).Height(maxHeight).Render(content)
	}
	return columnStyle.Width(width).Height(maxHeight).Render(content)
}

// formatAgentStatus returns a styled prefix based on agent status.
func formatAgentStatus(t sdk.TicketSummary) string {
	if t.AgentStatus == nil {
		return activeSessionStyle.Render("● ")
	}

	symbols := map[string]string{
		"starting":           "▶ ",
		"in_progress":        "● ",
		"idle":               "○ ",
		"waiting_permission": "⏸ ",
		"error":              "✗ ",
	}

	symbol := symbols[*t.AgentStatus]
	if symbol == "" {
		symbol = "● "
	}

	if t.AgentTool != nil && *t.AgentTool != "" {
		tool := *t.AgentTool
		if len(tool) > 8 {
			tool = tool[:8]
		}
		return activeSessionStyle.Render(symbol + tool + " ")
	}

	return activeSessionStyle.Render(symbol)
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
