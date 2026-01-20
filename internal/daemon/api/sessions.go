package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kareemaly/cortex1/internal/ticket"
	"github.com/kareemaly/cortex1/internal/tmux"
)

// SessionHandlers provides HTTP handlers for session operations.
type SessionHandlers struct {
	deps *Dependencies
}

// NewSessionHandlers creates a new SessionHandlers.
func NewSessionHandlers(deps *Dependencies) *SessionHandlers {
	return &SessionHandlers{deps: deps}
}

// Kill handles DELETE /sessions/{id} - kills a session.
func (h *SessionHandlers) Kill(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")

	// Search all tickets for the session
	ticketID, session := h.findSession(sessionID)
	if ticketID == "" {
		writeError(w, http.StatusNotFound, "not_found", "session not found")
		return
	}

	// If session is active and tmux is available, kill the window
	if session.IsActive() && h.deps.TmuxManager != nil {
		sessionName := h.deps.ProjectConfig.Name
		if sessionName == "" {
			sessionName = "cortex"
		}

		err := h.deps.TmuxManager.KillWindow(sessionName, session.TmuxWindow)
		if err != nil {
			// Log but don't fail - window might already be closed
			if !tmux.IsWindowNotFound(err) && !tmux.IsSessionNotFound(err) {
				h.deps.Logger.Warn("failed to kill tmux window", "error", err)
			}
		}
	}

	// End the session in the store
	if err := h.deps.TicketStore.EndSession(ticketID, sessionID); err != nil {
		if ticket.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "not_found", "session not found")
			return
		}
		h.deps.Logger.Error("failed to end session", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to end session")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// findSession searches all tickets for a session by ID.
// Returns the ticket ID and session, or empty string and nil if not found.
func (h *SessionHandlers) findSession(sessionID string) (string, *ticket.Session) {
	all, err := h.deps.TicketStore.ListAll()
	if err != nil {
		h.deps.Logger.Error("failed to list tickets", "error", err)
		return "", nil
	}

	for _, tickets := range all {
		for _, t := range tickets {
			for i := range t.Sessions {
				if t.Sessions[i].ID == sessionID {
					return t.ID, &t.Sessions[i]
				}
			}
		}
	}

	return "", nil
}
