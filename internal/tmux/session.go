package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// IsInsideTmux returns true if running inside a tmux session.
func IsInsideTmux() bool {
	return os.Getenv("TMUX") != ""
}

// SessionExists checks if a tmux session with the given name exists.
func (m *Manager) SessionExists(name string) (bool, error) {
	err := m.runSilent("has-session", "-t", sessionTarget(name)+":")
	if err != nil {
		// tmux returns exit code 1 if session doesn't exist
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CreateSession creates a new tmux session with the given name.
// If the session already exists, this is a no-op.
// The session is created detached (-d flag).
// If workingDir is specified, the session starts in that directory.
func (m *Manager) CreateSession(name, workingDir string) error {
	exists, err := m.SessionExists(name)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	args := []string{"new-session", "-d", "-s", name}
	if workingDir != "" {
		args = append(args, "-c", workingDir)
	}
	output, err := m.run(args...)
	if err != nil {
		return &CommandError{Command: "new-session", Output: strings.TrimSpace(string(output))}
	}
	return nil
}

// KillSession terminates a tmux session.
// Returns SessionNotFoundError if the session doesn't exist.
func (m *Manager) KillSession(name string) error {
	exists, err := m.SessionExists(name)
	if err != nil {
		return err
	}
	if !exists {
		return &SessionNotFoundError{Session: name}
	}

	output, err := m.run("kill-session", "-t", sessionTarget(name)+":")
	if err != nil {
		return &CommandError{Command: "kill-session", Output: strings.TrimSpace(string(output))}
	}
	return nil
}

// AttachSession attaches to an existing tmux session interactively.
// If already inside tmux, it uses switch-client instead of attach-session.
// Returns SessionNotFoundError if the session doesn't exist.
func (m *Manager) AttachSession(name string) error {
	exists, err := m.SessionExists(name)
	if err != nil {
		return err
	}
	if !exists {
		return &SessionNotFoundError{Session: name}
	}

	// Select window 0 (architect window) before attaching/switching
	target := fmt.Sprintf("%s:%d", sessionTarget(name), ArchitectWindowIndex)
	_, _ = m.run("select-window", "-t", target) // Non-fatal: window selection failing shouldn't prevent attach

	// Choose command based on whether we're inside tmux
	if IsInsideTmux() {
		return m.runner.RunInteractive("switch-client", "-t", sessionTarget(name)+":")
	}
	return m.runner.RunInteractive("attach-session", "-t", sessionTarget(name)+":")
}

// SwitchClient switches the most recently active tmux client to the given session.
// This is a non-interactive command safe for daemon use.
func (m *Manager) SwitchClient(session string) error {
	output, err := m.run("switch-client", "-t", sessionTarget(session)+":")
	if err != nil {
		return &CommandError{Command: "switch-client", Output: strings.TrimSpace(string(output))}
	}
	return nil
}
