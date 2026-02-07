package spawn

import (
	"github.com/kareemaly/cortex/internal/session"
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
)

// StateInfo contains information about a ticket's session state.
type StateInfo struct {
	State        SessionState
	Session      *session.Session
	WindowExists bool
}

// TmuxChecker provides the ability to check tmux window existence.
type TmuxChecker interface {
	WindowExists(session, windowName string) (bool, error)
}

// DetectTicketState determines the current state of a ticket's session.
// If sess is nil, returns StateNormal.
func DetectTicketState(sess *session.Session, tmuxSession string, tmuxChecker TmuxChecker) (*StateInfo, error) {
	info := &StateInfo{
		State:   StateNormal,
		Session: sess,
	}

	// No session - normal state
	if sess == nil {
		return info, nil
	}

	// Session exists - check if window is still running
	if tmuxChecker != nil && tmuxSession != "" && sess.TmuxWindow != "" {
		exists, err := tmuxChecker.WindowExists(tmuxSession, sess.TmuxWindow)
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
		// Can't check tmux - assume active if session exists
		info.State = StateActive
		info.WindowExists = true
	}

	return info, nil
}

// DetectArchitectState determines the current state of the architect session.
// Unlike DetectTicketState, this also handles the case where no session record
// exists but a tmux window named "architect" is present (pre-migration window).
func DetectArchitectState(sess *session.Session, tmuxSession string, tmuxChecker TmuxChecker) (*StateInfo, error) {
	info := &StateInfo{
		State:   StateNormal,
		Session: sess,
	}

	// Check if architect tmux window exists
	windowExists := false
	if tmuxChecker != nil && tmuxSession != "" {
		exists, err := tmuxChecker.WindowExists(tmuxSession, "architect")
		if err != nil {
			return nil, &TmuxError{Operation: "check architect window", Cause: err}
		}
		windowExists = exists
		info.WindowExists = exists
	}

	if sess != nil {
		// Session record exists — check if window is still running
		if windowExists {
			info.State = StateActive
		} else {
			info.State = StateOrphaned
		}
	} else if windowExists {
		// No session record but window exists (pre-migration or manual window).
		// Treat as active — we can't orphan-detect without a session record.
		info.State = StateActive
	}

	return info, nil
}

// CanSpawn returns true if a new session can be spawned based on the state.
func (s *StateInfo) CanSpawn() bool {
	return s.State == StateNormal || s.State == StateOrphaned
}

// CanResume returns true if an existing session can be resumed.
func (s *StateInfo) CanResume() bool {
	return s.State == StateOrphaned && s.Session != nil
}

// NeedsCleanup returns true if the session should be cleaned up before spawning.
func (s *StateInfo) NeedsCleanup() bool {
	return s.State == StateOrphaned
}
