package tmux

import (
	"fmt"
	"strconv"
	"strings"
)

// Window represents a tmux window.
type Window struct {
	Index  int
	Name   string
	Active bool
}

// WindowExists checks if a window with the given name exists in the session.
func (m *Manager) WindowExists(session, windowName string) (bool, error) {
	_, err := m.GetWindowByName(session, windowName)
	if err != nil {
		if IsWindowNotFound(err) || IsSessionNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CreateWindow creates a new window in the session with the given name.
// Returns the window index.
// If workingDir is specified, the window starts in that directory.
func (m *Manager) CreateWindow(session, windowName, workingDir string) (int, error) {
	exists, err := m.SessionExists(session)
	if err != nil {
		return 0, err
	}
	if !exists {
		return 0, &SessionNotFoundError{Session: session}
	}

	// Create window and capture its index.
	// -a: insert at the next index after the target, not at the target index itself.
	// Without -a, tmux resolves -t to the current window and tries to create AT that index,
	// which fails with "index N in use" if the window already exists.
	//
	// Use "session:" (with trailing colon) to force tmux to resolve the target as a session name.
	// Without the colon, tmux treats -t as a target-window and may prefix-match a window name
	// in the current session (e.g., "cortex" matches "cortexd"), creating the window in the
	// wrong session when the daemon runs inside a different tmux session.
	sessionTarget := fmt.Sprintf("%s:", session)
	args := []string{"new-window", "-a", "-t", sessionTarget, "-n", windowName, "-P", "-F", "#{window_index}"}
	if workingDir != "" {
		args = append(args, "-c", workingDir)
	}
	output, err := m.run(args...)
	if err != nil {
		return 0, &CommandError{Command: "new-window", Output: strings.TrimSpace(string(output))}
	}

	index, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return 0, fmt.Errorf("parse window index: %w", err)
	}

	return index, nil
}

// CreateArchitectWindow renames window 0 to the given name.
// This is used to set up the architect window in a new session.
func (m *Manager) CreateArchitectWindow(session, windowName string) error {
	exists, err := m.SessionExists(session)
	if err != nil {
		return err
	}
	if !exists {
		return &SessionNotFoundError{Session: session}
	}

	target := fmt.Sprintf("%s:%d", session, ArchitectWindowIndex)
	output, err := m.run("rename-window", "-t", target, windowName)
	if err != nil {
		return &CommandError{Command: "rename-window", Output: strings.TrimSpace(string(output))}
	}
	return nil
}

// KillWindow terminates a window by name.
func (m *Manager) KillWindow(session, windowName string) error {
	window, err := m.GetWindowByName(session, windowName)
	if err != nil {
		return err
	}
	return m.KillWindowByIndex(session, window.Index)
}

// KillWindowByIndex terminates a window by index.
func (m *Manager) KillWindowByIndex(session string, index int) error {
	exists, err := m.SessionExists(session)
	if err != nil {
		return err
	}
	if !exists {
		return &SessionNotFoundError{Session: session}
	}

	target := fmt.Sprintf("%s:%d", session, index)
	output, err := m.run("kill-window", "-t", target)
	if err != nil {
		return &CommandError{Command: "kill-window", Output: strings.TrimSpace(string(output))}
	}
	return nil
}

// FocusWindowByIndex selects a window by index.
func (m *Manager) FocusWindowByIndex(session string, index int) error {
	exists, err := m.SessionExists(session)
	if err != nil {
		return err
	}
	if !exists {
		return &SessionNotFoundError{Session: session}
	}

	target := fmt.Sprintf("%s:%d", session, index)
	output, err := m.run("select-window", "-t", target)
	if err != nil {
		return &CommandError{Command: "select-window", Output: strings.TrimSpace(string(output))}
	}
	return nil
}

// FocusWindow selects a window by name.
func (m *Manager) FocusWindow(session, windowName string) error {
	window, err := m.GetWindowByName(session, windowName)
	if err != nil {
		return err
	}

	target := fmt.Sprintf("%s:%d", session, window.Index)
	output, err := m.run("select-window", "-t", target)
	if err != nil {
		return &CommandError{Command: "select-window", Output: strings.TrimSpace(string(output))}
	}
	return nil
}

// ListWindows returns all windows in a session.
func (m *Manager) ListWindows(session string) ([]Window, error) {
	exists, err := m.SessionExists(session)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, &SessionNotFoundError{Session: session}
	}

	// Format: index:name:active_flag
	output, err := m.run("list-windows", "-t", session, "-F", "#{window_index}:#{window_name}:#{window_active}")
	if err != nil {
		return nil, &CommandError{Command: "list-windows", Output: strings.TrimSpace(string(output))}
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	windows := make([]Window, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 3)
		if len(parts) != 3 {
			continue
		}

		index, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		windows = append(windows, Window{
			Index:  index,
			Name:   parts[1],
			Active: parts[2] == "1",
		})
	}

	return windows, nil
}

// GetActiveWindow returns the currently active window in a session.
func (m *Manager) GetActiveWindow(session string) (*Window, error) {
	windows, err := m.ListWindows(session)
	if err != nil {
		return nil, err
	}

	for i := range windows {
		if windows[i].Active {
			return &windows[i], nil
		}
	}

	return nil, &WindowNotFoundError{Session: session, Window: "active"}
}

// GetWindowByName finds a window by name in a session.
func (m *Manager) GetWindowByName(session, windowName string) (*Window, error) {
	windows, err := m.ListWindows(session)
	if err != nil {
		return nil, err
	}

	for i := range windows {
		if windows[i].Name == windowName {
			return &windows[i], nil
		}
	}

	return nil, &WindowNotFoundError{Session: session, Window: windowName}
}
