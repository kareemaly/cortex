package tmux

import "fmt"

// NotInstalledError indicates tmux is not installed or not available.
type NotInstalledError struct{}

func (e *NotInstalledError) Error() string {
	return "tmux is not installed"
}

// SessionNotFoundError indicates the specified tmux session does not exist.
type SessionNotFoundError struct {
	Session string
}

func (e *SessionNotFoundError) Error() string {
	return fmt.Sprintf("tmux session not found: %s", e.Session)
}

// WindowNotFoundError indicates the specified window does not exist.
type WindowNotFoundError struct {
	Session string
	Window  string
}

func (e *WindowNotFoundError) Error() string {
	return fmt.Sprintf("tmux window not found: %s:%s", e.Session, e.Window)
}

// CommandError indicates a tmux command failed.
type CommandError struct {
	Command string
	Output  string
}

func (e *CommandError) Error() string {
	if e.Output != "" {
		return fmt.Sprintf("tmux command failed: %s: %s", e.Command, e.Output)
	}
	return fmt.Sprintf("tmux command failed: %s", e.Command)
}

// IsNotInstalled returns true if err is a NotInstalledError.
func IsNotInstalled(err error) bool {
	_, ok := err.(*NotInstalledError)
	return ok
}

// IsSessionNotFound returns true if err is a SessionNotFoundError.
func IsSessionNotFound(err error) bool {
	_, ok := err.(*SessionNotFoundError)
	return ok
}

// IsWindowNotFound returns true if err is a WindowNotFoundError.
func IsWindowNotFound(err error) bool {
	_, ok := err.(*WindowNotFoundError)
	return ok
}
