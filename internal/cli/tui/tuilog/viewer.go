package tuilog

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DismissLogViewerMsg signals that the log viewer should be closed.
type DismissLogViewerMsg struct{}

// Viewer is a Bubbletea sub-model for displaying log entries.
type Viewer struct {
	buf          *Buffer
	width        int
	height       int
	scrollOffset int
	minLevel     Level // filter: show entries >= this level
}

// NewViewer creates a new log viewer reading from the given buffer.
func NewViewer(buf *Buffer) Viewer {
	return Viewer{
		buf:      buf,
		minLevel: LevelDebug,
	}
}

// SetSize updates the viewer dimensions.
func (v *Viewer) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// Reset resets scroll position and filter when opening.
func (v *Viewer) Reset() {
	v.scrollOffset = 0
	v.minLevel = LevelDebug
}

// Update handles keyboard input for the viewer.
func (v Viewer) Update(msg tea.Msg) (Viewer, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return v, nil
	}

	switch keyMsg.String() {
	case "!", "esc":
		return v, func() tea.Msg { return DismissLogViewerMsg{} }

	case "j", "down":
		v.scrollOffset++
		return v, nil

	case "k", "up":
		if v.scrollOffset > 0 {
			v.scrollOffset--
		}
		return v, nil

	case "ctrl+d":
		v.scrollOffset += 10
		return v, nil

	case "ctrl+u":
		v.scrollOffset -= 10
		if v.scrollOffset < 0 {
			v.scrollOffset = 0
		}
		return v, nil

	case "1":
		v.minLevel = LevelDebug
		v.scrollOffset = 0
		return v, nil

	case "2":
		v.minLevel = LevelInfo
		v.scrollOffset = 0
		return v, nil

	case "3":
		v.minLevel = LevelWarn
		v.scrollOffset = 0
		return v, nil

	case "4":
		v.minLevel = LevelError
		v.scrollOffset = 0
		return v, nil
	}

	return v, nil
}

// View renders the log viewer as a full-screen overlay.
func (v Viewer) View() string {
	var b strings.Builder

	// Title bar.
	title := titleBarStyle.Render(" Logs ")
	filterInfo := filterInfoStyle.Render(fmt.Sprintf(" Filter: %s+ ", v.minLevel.String()))
	titlePadding := max(v.width-lipgloss.Width(title)-lipgloss.Width(filterInfo), 0)
	b.WriteString(title + strings.Repeat(" ", titlePadding) + filterInfo)
	b.WriteString("\n")

	// Content area: height minus title bar (1) and help bar (1).
	contentHeight := max(v.height-2, 1)

	// Get filtered entries.
	allEntries := v.buf.Entries()
	var filtered []Entry
	for _, e := range allEntries {
		if e.Level >= v.minLevel {
			filtered = append(filtered, e)
		}
	}

	// Clamp scroll offset.
	maxOffset := max(len(filtered)-contentHeight, 0)
	if v.scrollOffset > maxOffset {
		v.scrollOffset = maxOffset
	}

	// Render visible entries.
	start := v.scrollOffset
	end := min(start+contentHeight, len(filtered))

	lines := 0
	for i := start; i < end; i++ {
		e := filtered[i]
		line := v.renderEntry(e)
		b.WriteString(line)
		lines++
		if lines < contentHeight {
			b.WriteString("\n")
		}
	}

	// Fill remaining lines.
	for lines < contentHeight {
		lines++
		if lines < contentHeight {
			b.WriteString("\n")
		}
	}

	// Help bar.
	b.WriteString("\n")
	help := helpStyle.Render("[j/k] scroll  [ctrl+d/u] page  [1]all [2]info+ [3]warn+ [4]error  [!/esc] close")
	b.WriteString(help)

	return b.String()
}

// renderEntry renders a single log entry line.
func (v Viewer) renderEntry(e Entry) string {
	ts := timestampStyle.Render(e.Time.Format("15:04:05"))
	lvl := levelStyle(e.Level).Render(e.Level.ShortString())
	src := sourceStyle.Render(e.Source)

	// Truncate message to fit.
	prefix := fmt.Sprintf("%s %s %-6s ", e.Time.Format("15:04:05"), e.Level.ShortString(), e.Source)
	maxMsg := max(v.width-len(prefix)-1, 10)
	msg := e.Message
	if len(msg) > maxMsg {
		msg = msg[:maxMsg-3] + "..."
	}

	return fmt.Sprintf("%s %s %s %s", ts, lvl, src, msg)
}

// --- Styles ---

var (
	titleBarStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("124")) // dark red

	filterInfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("124"))

	timestampStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")) // gray

	sourceStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("33")). // blue
			Width(6)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
)

// levelStyle returns the style for a given log level.
func levelStyle(level Level) lipgloss.Style {
	switch level {
	case LevelError:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196")) // red
	case LevelWarn:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214")) // orange
	case LevelInfo:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("82")) // green
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // gray
	}
}
