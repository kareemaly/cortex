package tmux

import (
	"os"
	"os/exec"
	"strings"
)

// SessionExists checks if a tmux session with the given name exists.
func (m *Manager) SessionExists(name string) (bool, error) {
	err := m.runSilent("has-session", "-t", name)
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
func (m *Manager) CreateSession(name string) error {
	exists, err := m.SessionExists(name)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	output, err := m.run("new-session", "-d", "-s", name)
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

	output, err := m.run("kill-session", "-t", name)
	if err != nil {
		return &CommandError{Command: "kill-session", Output: strings.TrimSpace(string(output))}
	}
	return nil
}

// AttachSession attaches to an existing tmux session interactively.
// This replaces the current process with tmux attach-session.
// Returns SessionNotFoundError if the session doesn't exist.
func (m *Manager) AttachSession(name string) error {
	exists, err := m.SessionExists(name)
	if err != nil {
		return err
	}
	if !exists {
		return &SessionNotFoundError{Session: name}
	}

	// Use syscall exec to replace current process
	cmd := exec.Command(m.tmuxPath, "attach-session", "-t", name)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
