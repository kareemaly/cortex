package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	projectconfig "github.com/kareemaly/cortex/internal/project/config"
	"github.com/kareemaly/cortex/internal/prompt"
	"github.com/kareemaly/cortex/internal/session"
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

// List handles GET /sessions - lists all active sessions.
func (h *SessionHandlers) List(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())

	if h.deps.SessionManager == nil {
		writeJSON(w, http.StatusOK, map[string]any{"sessions": []any{}})
		return
	}

	sessStore := h.deps.SessionManager.GetStore(projectPath)
	sessions, err := sessStore.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "session_error", err.Error())
		return
	}

	// Resolve ticket titles
	ticketStore, _ := h.deps.StoreManager.GetStore(projectPath)

	type sessionListItem struct {
		SessionID   string    `json:"session_id"`
		TicketID    string    `json:"ticket_id"`
		TicketTitle string    `json:"ticket_title"`
		Agent       string    `json:"agent"`
		TmuxWindow  string    `json:"tmux_window"`
		StartedAt   time.Time `json:"started_at"`
		Status      string    `json:"status"`
		Tool        *string   `json:"tool,omitempty"`
	}

	items := make([]sessionListItem, 0, len(sessions))
	for shortID, sess := range sessions {
		title := ""
		if ticketStore != nil {
			if t, _, err := ticketStore.Get(sess.TicketID); err == nil {
				title = t.Title
			}
		}
		items = append(items, sessionListItem{
			SessionID:   shortID,
			TicketID:    sess.TicketID,
			TicketTitle: title,
			Agent:       sess.Agent,
			TmuxWindow:  sess.TmuxWindow,
			StartedAt:   sess.StartedAt,
			Status:      string(sess.Status),
			Tool:        sess.Tool,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"sessions": items,
		"total":    len(items),
	})
}

// Kill handles DELETE /sessions/{id} - kills a session.
// The {id} parameter is the ticket short ID (first 8 chars of the ticket ID).
func (h *SessionHandlers) Kill(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")

	projectPath := GetProjectPath(r.Context())

	sessStore := h.deps.SessionManager.GetStore(projectPath)

	// Search all sessions for a match
	shortID, sess := h.findSession(sessStore, sessionID)
	if sess == nil {
		writeError(w, http.StatusNotFound, "not_found", "session not found")
		return
	}

	// If session is active and tmux is available, kill the window
	if h.deps.TmuxManager != nil && sess.TmuxWindow != "" {
		// Load project config for session name
		projectCfg, err := projectconfig.Load(projectPath)
		sessionName := "cortex"
		if err == nil && projectCfg.Name != "" {
			sessionName = projectCfg.Name
		}

		err = h.deps.TmuxManager.KillWindow(sessionName, sess.TmuxWindow)
		if err != nil {
			// Log but don't fail - window might already be closed
			if !tmux.IsWindowNotFound(err) && !tmux.IsSessionNotFound(err) {
				h.deps.Logger.Warn("failed to kill tmux window", "error", err)
			}
		}
	}

	// End the session in the store
	if err := sessStore.End(shortID); err != nil {
		h.deps.Logger.Error("failed to end session", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to end session")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// findSession searches all sessions for one matching the given ID.
// The ID can be a ticket short ID or a ticket full ID (prefix match).
// Returns the short ID key and session, or ("", nil) if not found.
func (h *SessionHandlers) findSession(sessStore *session.Store, id string) (string, *session.Session) {
	sessions, _ := sessStore.List()
	for shortID, sess := range sessions {
		// Match by ticket short ID
		if shortID == id {
			return shortID, sess
		}
		// Match by ticket full ID prefix
		if len(sess.TicketID) >= len(id) && sess.TicketID[:len(id)] == id {
			return shortID, sess
		}
	}
	return "", nil
}

// Approve handles POST /sessions/{id}/approve - sends approve prompt to agent.
func (h *SessionHandlers) Approve(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")

	projectPath := GetProjectPath(r.Context())

	sessStore := h.deps.SessionManager.GetStore(projectPath)

	// Find the session
	_, sess := h.findSession(sessStore, sessionID)
	if sess == nil {
		writeError(w, http.StatusNotFound, "not_found", "session not found")
		return
	}

	// Get the ticket for prompt rendering
	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	t, _, err := store.Get(sess.TicketID)
	if err != nil {
		if ticket.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "not_found", "ticket not found for session")
			return
		}
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
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

	// Load and render approve prompt with fallback support
	ticketType := t.Type
	if ticketType == "" {
		ticketType = ticket.DefaultTicketType
	}
	resolver := prompt.NewPromptResolver(projectPath, projectCfg.ResolvedExtendPath())
	approveContent, err := resolver.ResolveTicketPrompt(ticketType, prompt.StageApprove)
	if err != nil {
		// Use a default message if file doesn't exist
		approveContent = "Your changes have been approved. Please call `mcp__cortex__concludeSession` with a full report to complete this ticket."
	}

	// Render template variables
	vars := prompt.TicketVars{
		ProjectPath: projectPath,
		TicketID:    t.ID,
		TicketTitle: t.Title,
		TicketBody:  t.Body,
		IsWorktree:  sess.WorktreePath != nil,
	}
	if sess.WorktreePath != nil {
		vars.WorktreePath = *sess.WorktreePath
	}
	if sess.FeatureBranch != nil {
		vars.WorktreeBranch = *sess.FeatureBranch
	}
	rendered, err := prompt.RenderTemplate(approveContent, vars)
	if err != nil {
		h.deps.Logger.Warn("failed to render approve template", "error", err)
		// Fall through with unrendered content
	} else {
		approveContent = rendered
	}

	// Get window info from session
	window, err := h.deps.TmuxManager.GetWindowByName(tmuxSession, sess.TmuxWindow)
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

	// Focus the tmux window (non-fatal if this fails)
	if err := h.deps.TmuxManager.FocusWindow(tmuxSession, sess.TmuxWindow); err != nil {
		h.deps.Logger.Warn("failed to focus tmux window", "error", err)
	}

	if err := h.deps.TmuxManager.SwitchClient(tmuxSession); err != nil {
		h.deps.Logger.Warn("failed to switch tmux client", "session", tmuxSession, "error", err)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"session_id": sessionID,
		"message":    "Approve prompt sent to agent",
	})
}
