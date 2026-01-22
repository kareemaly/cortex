package spawn

import (
	"github.com/kareemaly/cortex/internal/ticket"
)

// SessionState represents the state of a ticket's agent session.
type SessionState string

const (
	// StateNormal indicates no active session exists.
	StateNormal SessionState = "normal"
	// StateActive indicates a session exists and the tmux window is running.
	StateActive SessionState = "active"
	// StateOrphaned indicates a session exists but the tmux window is gone.
	StateOrphaned SessionState = "orphaned"
	// StateEnded indicates the session has been explicitly ended.
	StateEnded SessionState = "ended"
)

// StateInfo contains information about a ticket's session state.
type StateInfo struct {
	State           SessionState
	Session         *ticket.Session
	WindowExists    bool
	ClaudeSessionID string
}

// TmuxChecker provides the ability to check tmux window existence.
type TmuxChecker interface {
	WindowExists(session, windowName string) (bool, error)
}

// DetectTicketState determines the current state of a ticket's session.
func DetectTicketState(t *ticket.Ticket, tmuxSession string, tmuxChecker TmuxChecker) (*StateInfo, error) {
	info := &StateInfo{
		State:   StateNormal,
		Session: t.Session,
	}

	// No session - normal state
	if t.Session == nil {
		return info, nil
	}

	info.ClaudeSessionID = t.Session.ClaudeSessionID

	// Session ended explicitly
	if t.Session.EndedAt != nil {
		info.State = StateEnded
		return info, nil
	}

	// Session exists - check if window is still running
	if tmuxChecker != nil && tmuxSession != "" && t.Session.TmuxWindow != "" {
		exists, err := tmuxChecker.WindowExists(tmuxSession, t.Session.TmuxWindow)
		if err != nil {
			return nil, &TmuxError{Operation: "check window", Cause: err}
		}
		info.WindowExists = exists

		if exists {
			info.State = StateActive
		} else {
			info.State = StateOrphaned
		}
	} else {
		// Can't check tmux - assume active if session hasn't ended
		info.State = StateActive
		info.WindowExists = true
	}

	return info, nil
}

// CanSpawn returns true if a new session can be spawned based on the state.
func (s *StateInfo) CanSpawn() bool {
	return s.State == StateNormal || s.State == StateOrphaned || s.State == StateEnded
}

// CanResume returns true if an existing session can be resumed.
func (s *StateInfo) CanResume() bool {
	return s.State == StateOrphaned && s.ClaudeSessionID != ""
}

// NeedsCleanup returns true if the session should be cleaned up before spawning.
func (s *StateInfo) NeedsCleanup() bool {
	return s.State == StateOrphaned || s.State == StateEnded
}
