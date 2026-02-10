package tmux

import (
	"strconv"
	"strings"
)

// Client represents a tmux client attached to a session.
type Client struct {
	TTY        string // e.g., "/dev/ttys000"
	Session    string // Session name
	Window     int    // Current window index
	WindowName string // Current window name
}

// ListClients returns all clients attached to the specified session.
// Returns an empty slice (not error) when no clients are attached.
// Returns SessionNotFoundError for nonexistent sessions.
func (m *Manager) ListClients(session string) ([]Client, error) {
	// First check if session exists
	exists, err := m.SessionExists(session)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, &SessionNotFoundError{Session: session}
	}

	// Format: tty:session:window_index:window_name
	output, err := m.run("list-clients", "-t", session+":", "-F", "#{client_tty}:#{client_session}:#{window_index}:#{window_name}")
	if err != nil {
		// list-clients returns error if no clients, but we want empty slice
		return []Client{}, nil
	}

	lines := strings.TrimSpace(string(output))
	if lines == "" {
		return []Client{}, nil
	}

	var clients []Client
	for _, line := range strings.Split(lines, "\n") {
		if line == "" {
			continue
		}

		// Use SplitN to handle colons in window names
		parts := strings.SplitN(line, ":", 4)
		if len(parts) < 4 {
			continue
		}

		windowIdx, err := strconv.Atoi(parts[2])
		if err != nil {
			continue
		}

		clients = append(clients, Client{
			TTY:        parts[0],
			Session:    parts[1],
			Window:     windowIdx,
			WindowName: parts[3],
		})
	}

	return clients, nil
}

// IsUserAttached returns true if any client is viewing the specified window.
// Returns false on error (safe default for notifications).
func (m *Manager) IsUserAttached(session, windowName string) bool {
	clients, err := m.ListClients(session)
	if err != nil {
		return false
	}

	for _, c := range clients {
		if c.WindowName == windowName {
			return true
		}
	}
	return false
}

// IsUserAttachedToWindow returns true if any client is viewing the specified window index.
// Returns false on error (safe default for notifications).
func (m *Manager) IsUserAttachedToWindow(session string, windowIndex int) bool {
	clients, err := m.ListClients(session)
	if err != nil {
		return false
	}

	for _, c := range clients {
		if c.Window == windowIndex {
			return true
		}
	}
	return false
}
