package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	projectconfig "github.com/kareemaly/cortex/internal/project/config"
	"github.com/kareemaly/cortex/internal/prompt"
	"github.com/kareemaly/cortex/internal/ticket"
	"github.com/kareemaly/cortex/internal/tmux"
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
	if err := store.EndSession(ticketID); err != nil {
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
			if t.Session != nil && t.Session.ID == sessionID {
				return t.ID, t.Session
			}
		}
	}

	return "", nil
}

// findSessionWithTicket searches all tickets for a session by ID.
// Returns the ticket and session, or nil if not found.
func (h *SessionHandlers) findSessionWithTicket(store *ticket.Store, sessionID string) (*ticket.Ticket, *ticket.Session) {
	all, err := store.ListAll()
	if err != nil {
		h.deps.Logger.Error("failed to list tickets", "error", err)
		return nil, nil
	}

	for _, tickets := range all {
		for _, t := range tickets {
			if t.Session != nil && t.Session.ID == sessionID {
				return t, t.Session
			}
		}
	}

	return nil, nil
}

// Approve handles POST /sessions/{id}/approve - sends approve prompt to agent.
func (h *SessionHandlers) Approve(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")

	projectPath := GetProjectPath(r.Context())
	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	// Find the session and ticket
	t, session := h.findSessionWithTicket(store, sessionID)
	if t == nil {
		writeError(w, http.StatusNotFound, "not_found", "session not found")
		return
	}

	// Validate session is active
	if !session.IsActive() {
		writeError(w, http.StatusBadRequest, "session_inactive", "session is not active")
		return
	}

	// Check tmux manager is available
	if h.deps.TmuxManager == nil {
		writeError(w, http.StatusInternalServerError, "tmux_unavailable", "tmux is not available")
		return
	}

	// Load project config for session name
	projectCfg, err := projectconfig.Load(projectPath)
	tmuxSession := "cortex"
	if err == nil && projectCfg.Name != "" {
		tmuxSession = projectCfg.Name
	}

	// Load and render approve prompt
	approvePath := prompt.ApprovePath(projectPath)
	approveContent, err := prompt.LoadPromptFile(approvePath)
	if err != nil {
		// Use a default message if file doesn't exist
		approveContent = "Your changes have been approved. Please call `mcp__cortex__concludeSession` with a full report to complete this ticket."
	}

	// Get window info from session
	window, err := h.deps.TmuxManager.GetWindowByName(tmuxSession, session.TmuxWindow)
	if err != nil {
		if tmux.IsWindowNotFound(err) || tmux.IsSessionNotFound(err) {
			writeError(w, http.StatusNotFound, "window_not_found", "tmux window not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "tmux_error", err.Error())
		return
	}

	// Send approve prompt to agent pane (pane 0 is the left pane with Claude)
	if err := h.deps.TmuxManager.RunCommandInPane(tmuxSession, window.Index, 0, approveContent); err != nil {
		h.deps.Logger.Error("failed to send approve prompt", "error", err)
		writeError(w, http.StatusInternalServerError, "send_failed", "failed to send approve prompt to agent")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"session_id": sessionID,
		"message":    "Approve prompt sent to agent",
	})
}
