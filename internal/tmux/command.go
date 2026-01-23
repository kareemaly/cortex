package tmux

import (
	"fmt"
	"strings"
)

// RunCommand sends a command to a window and presses Enter.
func (m *Manager) RunCommand(session, windowName, command string) error {
	window, err := m.GetWindowByName(session, windowName)
	if err != nil {
		return err
	}
	return m.RunCommandInIndex(session, window.Index, command)
}

// RunCommandInIndex sends a command to a window by index and presses Enter.
func (m *Manager) RunCommandInIndex(session string, index int, command string) error {
	target := fmt.Sprintf("%s:%d", session, index)
	output, err := m.run("send-keys", "-t", target, command, "Enter")
	if err != nil {
		return &CommandError{Command: "send-keys", Output: strings.TrimSpace(string(output))}
	}
	return nil
}

// SendKeys sends key sequences to a window without pressing Enter.
// Useful for sending control sequences like C-c.
func (m *Manager) SendKeys(session, windowName string, keys ...string) error {
	window, err := m.GetWindowByName(session, windowName)
	if err != nil {
		return err
	}
	return m.SendKeysToIndex(session, window.Index, keys...)
}

// SendKeysToIndex sends key sequences to a window by index without pressing Enter.
func (m *Manager) SendKeysToIndex(session string, index int, keys ...string) error {
	target := fmt.Sprintf("%s:%d", session, index)
	args := append([]string{"send-keys", "-t", target}, keys...)
	output, err := m.run(args...)
	if err != nil {
		return &CommandError{Command: "send-keys", Output: strings.TrimSpace(string(output))}
	}
	return nil
}

// SpawnAgent creates a session (if needed), creates a new window, and runs the agent command.
// Returns the window index.
// If workingDir is specified, both the session and window start in that directory.
func (m *Manager) SpawnAgent(session, windowName, agentCommand, workingDir string) (int, error) {
	// Ensure session exists
	if err := m.CreateSession(session, workingDir); err != nil {
		return 0, err
	}

	// Create window for the agent
	index, err := m.CreateWindow(session, windowName, workingDir)
	if err != nil {
		return 0, err
	}

	// Run the agent command
	if err := m.RunCommandInIndex(session, index, agentCommand); err != nil {
		return index, err
	}

	return index, nil
}

// SpawnArchitect creates a session (if needed), renames window 0, and runs the architect command.
// Window 0 is reserved for the architect agent.
// If workingDir is specified, the session starts in that directory.
// For existing sessions, the command is prefixed with a cd to ensure correct directory.
func (m *Manager) SpawnArchitect(session, windowName, agentCommand, workingDir string) error {
	// Check if session exists first
	exists, err := m.SessionExists(session)
	if err != nil {
		return err
	}

	if !exists {
		if err := m.CreateSession(session, workingDir); err != nil {
			return err
		}
	}

	// Rename window 0 for the architect
	if err := m.CreateArchitectWindow(session, windowName); err != nil {
		return err
	}

	// If session existed, prepend cd to ensure correct directory
	cmdToRun := agentCommand
	if exists && workingDir != "" {
		cmdToRun = fmt.Sprintf("cd %q && %s", workingDir, agentCommand)
	}

	// Run the architect command
	return m.RunCommandInIndex(session, ArchitectWindowIndex, cmdToRun)
}
