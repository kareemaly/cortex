package tmux

import (
	"fmt"
	"strings"
)

// SplitWindowHorizontal splits a window horizontally (side-by-side) creating a new pane on the right.
// The new pane starts in the specified working directory.
func (m *Manager) SplitWindowHorizontal(session string, windowIndex int, workingDir string) error {
	target := fmt.Sprintf("%s:%d", session, windowIndex)
	args := []string{"split-window", "-h", "-p", "70", "-t", target}
	if workingDir != "" {
		args = append(args, "-c", workingDir)
	}
	output, err := m.run(args...)
	if err != nil {
		return &CommandError{Command: "split-window", Output: strings.TrimSpace(string(output))}
	}
	return nil
}

// ResetWindowPanes kills all panes in a window except pane 0, restoring it to a single-pane layout.
// This is a no-op if the window already has only one pane.
func (m *Manager) ResetWindowPanes(session string, windowIndex int) error {
	target := fmt.Sprintf("%s:%d.0", session, windowIndex)
	// kill-pane -a kills all panes except the target; ignore errors (single pane case)
	_, _ = m.run("kill-pane", "-a", "-t", target)
	return nil
}

// RunCommandInPane sends a command to a specific pane within a window and presses Enter.
// paneIndex: 0=left, 1=right (after horizontal split)
func (m *Manager) RunCommandInPane(session string, windowIndex, paneIndex int, command string) error {
	target := fmt.Sprintf("%s:%d.%d", session, windowIndex, paneIndex)
	output, err := m.run("send-keys", "-t", target, command, "Enter")
	if err != nil {
		return &CommandError{Command: "send-keys", Output: strings.TrimSpace(string(output))}
	}
	return nil
}

// SelectPane selects (focuses) a specific pane within a window.
// paneIndex: 0=left, 1=right (after horizontal split)
func (m *Manager) SelectPane(session string, windowIndex, paneIndex int) error {
	target := fmt.Sprintf("%s:%d.%d", session, windowIndex, paneIndex)
	output, err := m.run("select-pane", "-t", target)
	if err != nil {
		return &CommandError{Command: "select-pane", Output: strings.TrimSpace(string(output))}
	}
	return nil
}
