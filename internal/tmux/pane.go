package tmux

import (
	"fmt"
	"strconv"
	"strings"
)

// splitHorizontal splits a window horizontally using an absolute cell count
// for the new right-hand pane. Using absolute cells (not -p N or -l N%) is
// the only form supported across every tmux version we care about:
//
//   - tmux 1.x: -l <absolute>                         ✓
//   - tmux 2.x: -l <absolute>, -p <percent>           ✓
//   - tmux 3.x: -l <absolute>, -l N%, -p N deprecated/rejected (no client → "size missing")
//
// We query the window width via display-message and compute the size from
// percent ourselves, so the behavior is deterministic regardless of whether
// a tmux client is attached.
func (m *Manager) splitHorizontal(target string, percent int, workingDir string) error {
	width, err := m.getWindowWidth(target)
	if err != nil {
		return err
	}
	size := width * percent / 100
	if size < 1 {
		size = 1
	}
	args := []string{"split-window", "-h", "-l", strconv.Itoa(size), "-t", target}
	if workingDir != "" {
		args = append(args, "-c", workingDir)
	}
	output, err := m.run(args...)
	if err != nil {
		return &CommandError{Command: "split-window", Output: strings.TrimSpace(string(output))}
	}
	return nil
}

// getWindowWidth returns the width (in cells) of the window at the given
// target. Uses display-message, which is supported in every tmux version
// back to 1.8. The window inherits its size from the detached session's
// -x/-y (see session.go CreateSession).
func (m *Manager) getWindowWidth(target string) (int, error) {
	output, err := m.run("display-message", "-p", "-F", "#{window_width}", "-t", target)
	if err != nil {
		return 0, &CommandError{Command: "display-message", Output: strings.TrimSpace(string(output))}
	}
	return strconv.Atoi(strings.TrimSpace(string(output)))
}

// SplitWindowHorizontal splits a window horizontally (side-by-side) creating a new pane on the right.
// The new pane starts in the specified working directory and occupies 70% of the window width.
func (m *Manager) SplitWindowHorizontal(session string, windowIndex int, workingDir string) error {
	target := fmt.Sprintf("%s:%d", sessionTarget(session), windowIndex)
	return m.splitHorizontal(target, 70, workingDir)
}

// SplitWindowHorizontalWithPercent splits a window horizontally (side-by-side) creating a new pane on the right.
// The new pane occupies the given percent of the window width.
func (m *Manager) SplitWindowHorizontalWithPercent(session string, windowIndex, percent int, workingDir string) error {
	target := fmt.Sprintf("%s:%d", sessionTarget(session), windowIndex)
	return m.splitHorizontal(target, percent, workingDir)
}

// ResetWindowPanes kills all panes in a window except pane 0, restoring it to a single-pane layout.
// This is a no-op if the window already has only one pane.
func (m *Manager) ResetWindowPanes(session string, windowIndex int) error {
	target := fmt.Sprintf("%s:%d.0", sessionTarget(session), windowIndex)
	// kill-pane -a kills all panes except the target; ignore errors (single pane case)
	_, _ = m.run("kill-pane", "-a", "-t", target)
	return nil
}

// RunCommandInPane sends a command to a specific pane within a window and presses Enter.
// paneIndex: 0=left, 1=right (after horizontal split)
func (m *Manager) RunCommandInPane(session string, windowIndex, paneIndex int, command string) error {
	target := fmt.Sprintf("%s:%d.%d", sessionTarget(session), windowIndex, paneIndex)
	output, err := m.run("send-keys", "-t", target, command, "Enter")
	if err != nil {
		return &CommandError{Command: "send-keys", Output: strings.TrimSpace(string(output))}
	}
	return nil
}

// SelectPane selects (focuses) a specific pane within a window.
// paneIndex: 0=left, 1=right (after horizontal split)
func (m *Manager) SelectPane(session string, windowIndex, paneIndex int) error {
	target := fmt.Sprintf("%s:%d.%d", sessionTarget(session), windowIndex, paneIndex)
	output, err := m.run("select-pane", "-t", target)
	if err != nil {
		return &CommandError{Command: "select-pane", Output: strings.TrimSpace(string(output))}
	}
	return nil
}
