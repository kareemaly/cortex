package kanban

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/kareemaly/cortex1/internal/cli/sdk"
)

// Column represents a kanban column with tickets.
type Column struct {
	title   string
	status  string
	tickets []sdk.TicketSummary
	cursor  int
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

// View renders the column.
func (c *Column) View(width int, isActive bool, maxHeight int) string {
	var b strings.Builder

	// Render header with count.
	headerText := fmt.Sprintf("%s (%d)", c.title, len(c.tickets))
	header := columnHeaderStyle(c.status).Width(width - 2).Render(headerText)
	b.WriteString(header)
	b.WriteString("\n")

	// Render tickets.
	if len(c.tickets) == 0 {
		emptyText := lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true).
			Render("(empty)")
		b.WriteString(emptyText)
		b.WriteString("\n")
	} else {
		for i, t := range c.tickets {
			// Truncate title if too long.
			title := t.Title
			maxLen := max(width-6, 10)
			if len(title) > maxLen {
				title = title[:maxLen-3] + "..."
			}

			// Build the line.
			var line string
			if i == c.cursor && isActive {
				line = selectedTicketStyle.Width(width - 2).Render("> " + title)
			} else {
				prefix := "  "
				if t.HasActiveSessions {
					prefix = activeSessionStyle.Render("● ")
				}
				line = ticketStyle.Width(width - 2).Render(prefix + title)
			}
			b.WriteString(line)
			if i < len(c.tickets)-1 {
				b.WriteString("\n")
			}
		}
	}

	content := b.String()

	// Apply column style.
	if isActive {
		return activeColumnStyle.Width(width).Height(maxHeight).Render(content)
	}
	return columnStyle.Width(width).Height(maxHeight).Render(content)
}
