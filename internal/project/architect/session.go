package architect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const stateFileName = "architect.json"

// Session represents an architect session's state.
type Session struct {
	ID          string     `json:"id"` // Used as Claude session ID for resume
	TmuxSession string     `json:"tmux_session"`
	TmuxWindow  string     `json:"tmux_window"`
	StartedAt   time.Time  `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at,omitempty"`
}

// IsActive returns true if the session has not ended.
func (s *Session) IsActive() bool {
	return s.EndedAt == nil
}

// Load reads the architect session state from the project's .cortex directory.
// Returns nil, nil if the state file does not exist.
func Load(projectPath string) (*Session, error) {
	statePath := filepath.Join(projectPath, ".cortex", stateFileName)

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// Save writes the architect session state to the project's .cortex directory.
// Returns an error if session is nil.
func Save(projectPath string, session *Session) error {
	if session == nil {
		return &NilSessionError{}
	}

	cortexDir := filepath.Join(projectPath, ".cortex")
	if err := os.MkdirAll(cortexDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}

	statePath := filepath.Join(cortexDir, stateFileName)
	return os.WriteFile(statePath, data, 0644)
}

// Clear removes the architect session state file.
// This operation is idempotent - no error is returned if the file doesn't exist.
func Clear(projectPath string) error {
	statePath := filepath.Join(projectPath, ".cortex", stateFileName)
	err := os.Remove(statePath)
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}

// NilSessionError is returned when Save is called with a nil session.
type NilSessionError struct{}

func (e *NilSessionError) Error() string {
	return "cannot save nil session"
}
