package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	architectconfig "github.com/kareemaly/cortex/internal/architect/config"
	"github.com/kareemaly/cortex/internal/events"
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
	projectPath := GetArchitectPath(r.Context())

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
		SessionType string    `json:"session_type"`
		TicketID    string    `json:"ticket_id"`
		TicketTitle string    `json:"ticket_title"`
		Agent       string    `json:"agent"`
		TmuxWindow  string    `json:"tmux_window"`
		StartedAt   time.Time `json:"started_at"`
		Status      string    `json:"status"`
		Tool        *string   `json:"tool,omitempty"`
	}

	items := make([]sessionListItem, 0, len(sessions))
	for _, sess := range sessions {
		sessionType := "ticket"
		title := ""
		switch sess.Type {
		case session.SessionTypeArchitect:
			sessionType = "architect"
			title = "Architect"
		case session.SessionTypeCollab:
			sessionType = "collab"
			if sess.Prompt != "" {
				if len(sess.Prompt) > 50 {
					title = "Collab: " + sess.Prompt[:47] + "..."
				} else {
					title = "Collab: " + sess.Prompt
				}
			} else {
				title = "Collab"
			}
		default:
			if ticketStore != nil {
				if t, _, err := ticketStore.Get(sess.TicketID); err == nil {
					title = t.Title
				}
			}
		}
		item := sessionListItem{
			SessionID:   sess.SessionID,
			SessionType: sessionType,
			TicketID:    sess.TicketID,
			TicketTitle: title,
			Agent:       sess.Agent,
			TmuxWindow:  sess.TmuxWindow,
			StartedAt:   sess.StartedAt,
			Status:      string(sess.Status),
			Tool:        sess.Tool,
		}

		// Overlay Hub-sourced status/tool if available.
		if h.deps.HubManager != nil && sess.AgentSessionID != "" {
			if ev, ok := h.deps.HubManager.GetEvent(sess.AgentSessionID); ok {
				item.Status = string(ev.Status)
				if ev.Tool != "" {
					tool := ev.Tool
					item.Tool = &tool
				} else {
					item.Tool = nil
				}
			}
		}

		items = append(items, item)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"sessions": items,
		"total":    len(items),
	})
}

// Kill handles DELETE /sessions/{id} - kills a session by its canonical
// SessionID UUID.
func (h *SessionHandlers) Kill(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")

	if h.deps.SessionManager == nil {
		writeError(w, http.StatusServiceUnavailable, "sessions_unavailable",
			"session manager is not configured")
		return
	}
	projectPath := GetArchitectPath(r.Context())

	sessStore := h.deps.SessionManager.GetStore(projectPath)

	sess, err := sessStore.GetBySessionID(sessionID)
	if err != nil || sess == nil {
		writeError(w, http.StatusNotFound, "not_found", "session not found")
		return
	}

	// If session is active and tmux is available, kill the window
	if h.deps.TmuxManager != nil && sess.TmuxWindow != "" {
		projectCfg, _ := architectconfig.Load(projectPath)
		sessionName := projectCfg.GetTmuxSessionName()

		if err := h.deps.TmuxManager.KillWindow(sessionName, sess.TmuxWindow); err != nil {
			if !tmux.IsWindowNotFound(err) && !tmux.IsSessionNotFound(err) {
				h.deps.Logger.Warn("failed to kill tmux window", "error", err)
			}
		}
	}

	// End the session in the store
	if err := sessStore.EndBySessionID(sess.SessionID); err != nil {
		h.deps.Logger.Error("failed to end session", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to end session")
		return
	}

	h.deps.Bus.Emit(events.Event{
		Type:          events.SessionEnded,
		ArchitectPath: projectPath,
		TicketID:      sess.TicketID,
		SessionID:     sess.SessionID,
	})

	w.WriteHeader(http.StatusNoContent)
}

// Approve handles POST /sessions/{id}/approve - sends approve prompt to agent.
func (h *SessionHandlers) Approve(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")

	if h.deps.SessionManager == nil {
		writeError(w, http.StatusServiceUnavailable, "sessions_unavailable",
			"session manager is not configured")
		return
	}
	projectPath := GetArchitectPath(r.Context())

	sessStore := h.deps.SessionManager.GetStore(projectPath)

	sess, err := sessStore.GetBySessionID(sessionID)
	if err != nil || sess == nil {
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
		writeError(w, http.StatusServiceUnavailable, "tmux_unavailable", "tmux is not installed")
		return
	}

	// Load project config for session name
	projectCfg, err := architectconfig.Load(projectPath)
	tmuxSession := "cortex"
	if err == nil && projectCfg.Name != "" {
		tmuxSession = projectCfg.Name
	}

	// Agents call concludeSession themselves — Approve just nudges them.
	approveContent := "Please call `concludeSession` with a summary of your work when you are done."

	// Render template variables
	vars := prompt.TicketVars{
		ProjectPath: projectPath,
		TicketID:    t.ID,
		TicketTitle: t.Title,
		TicketBody:  t.Body,
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

	writeJSON(w, http.StatusOK, map[string]any{
		"success":    true,
		"session_id": sessionID,
		"message":    "Approve prompt sent to agent",
	})
}
