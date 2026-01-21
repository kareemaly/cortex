package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	projectconfig "github.com/kareemaly/cortex1/internal/project/config"
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

	projectPath := GetProjectPath(r.Context())
	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	// Search all tickets for the session
	ticketID, session := h.findSession(store, sessionID)
	if ticketID == "" {
		writeError(w, http.StatusNotFound, "not_found", "session not found")
		return
	}

	// If session is active and tmux is available, kill the window
	if session.IsActive() && h.deps.TmuxManager != nil {
		// Load project config for session name
		projectCfg, err := projectconfig.Load(projectPath)
		sessionName := "cortex"
		if err == nil && projectCfg.Name != "" {
			sessionName = projectCfg.Name
		}

		err = h.deps.TmuxManager.KillWindow(sessionName, session.TmuxWindow)
		if err != nil {
			// Log but don't fail - window might already be closed
			if !tmux.IsWindowNotFound(err) && !tmux.IsSessionNotFound(err) {
				h.deps.Logger.Warn("failed to kill tmux window", "error", err)
			}
		}
	}

	// End the session in the store
	if err := store.EndSession(ticketID, sessionID); err != nil {
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
func (h *SessionHandlers) findSession(store *ticket.Store, sessionID string) (string, *ticket.Session) {
	all, err := store.ListAll()
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
