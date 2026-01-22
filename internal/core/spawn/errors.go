package spawn

import "fmt"

// StateError indicates an invalid session state for the requested operation.
type StateError struct {
	TicketID string
	State    SessionState
	Message  string
}

func (e *StateError) Error() string {
	return fmt.Sprintf("spawn: ticket %s in state %s: %s", e.TicketID, e.State, e.Message)
}

// ConfigError indicates missing or invalid configuration.
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("spawn: config error: %s: %s", e.Field, e.Message)
}

// TmuxError indicates a tmux operation failed.
type TmuxError struct {
	Operation string
	Cause     error
}

func (e *TmuxError) Error() string {
	return fmt.Sprintf("spawn: tmux %s: %s", e.Operation, e.Cause)
}

func (e *TmuxError) Unwrap() error {
	return e.Cause
}

// BinaryNotFoundError indicates cortexd binary was not found.
type BinaryNotFoundError struct {
	Binary string
	Cause  error
}

func (e *BinaryNotFoundError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("spawn: %s not found: %s", e.Binary, e.Cause)
	}
	return fmt.Sprintf("spawn: %s not found", e.Binary)
}

func (e *BinaryNotFoundError) Unwrap() error {
	return e.Cause
}

// PromptError indicates a prompt template loading failure.
type PromptError struct {
	AgentType AgentType
	Cause     error
}

func (e *PromptError) Error() string {
	return fmt.Sprintf("spawn: failed to load %s prompt: %s", e.AgentType, e.Cause)
}

func (e *PromptError) Unwrap() error {
	return e.Cause
}

// IsStateError returns true if err is a StateError.
func IsStateError(err error) bool {
	_, ok := err.(*StateError)
	return ok
}

// IsTmuxError returns true if err is a TmuxError.
func IsTmuxError(err error) bool {
	_, ok := err.(*TmuxError)
	return ok
}

// IsConfigError returns true if err is a ConfigError.
func IsConfigError(err error) bool {
	_, ok := err.(*ConfigError)
	return ok
}

// IsBinaryNotFoundError returns true if err is a BinaryNotFoundError.
func IsBinaryNotFoundError(err error) bool {
	_, ok := err.(*BinaryNotFoundError)
	return ok
}
