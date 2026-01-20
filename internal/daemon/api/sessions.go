package api

import (
	"net/http"
)

// SessionHandlers provides HTTP handlers for session operations.
type SessionHandlers struct{}

// NewSessionHandlers creates a new SessionHandlers.
func NewSessionHandlers() *SessionHandlers {
	return &SessionHandlers{}
}

// Kill handles DELETE /sessions/{id} - kills a session (stub).
func (h *SessionHandlers) Kill(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "kill session is not yet implemented")
}
