package api

import (
	"encoding/json"
	"net/http"

	"github.com/kareemaly/cortex/internal/events"
	"github.com/kareemaly/cortex/internal/session"
	"github.com/kareemaly/cortex/internal/storage"
	"github.com/kareemaly/cortex/internal/ticket"
)

// AgentHandlers provides HTTP handlers for agent operations.
type AgentHandlers struct {
	deps *Dependencies
}

// NewAgentHandlers creates a new AgentHandlers with the given dependencies.
func NewAgentHandlers(deps *Dependencies) *AgentHandlers {
	return &AgentHandlers{deps: deps}
}

// UpdateAgentStatusRequest is the request body for updating agent status.
type UpdateAgentStatusRequest struct {
	TicketID string  `json:"ticket_id"`
	Status   string  `json:"status"`
	Tool     *string `json:"tool,omitempty"`
	Work     *string `json:"work,omitempty"`
}

// UpdateStatus handles POST /agent/status - updates the agent's current status.
func (h *AgentHandlers) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	var req UpdateAgentStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	if req.TicketID == "" {
		writeError(w, http.StatusBadRequest, "missing_ticket_id", "ticket_id is required")
		return
	}

	if req.Status == "" {
		writeError(w, http.StatusBadRequest, "missing_status", "status is required")
		return
	}

	// Validate status is a known agent status
	agentStatus := session.AgentStatus(req.Status)
	if !validAgentStatus(agentStatus) {
		writeError(w, http.StatusBadRequest, "invalid_status", "invalid agent status")
		return
	}

	projectPath := GetProjectPath(r.Context())

	sessStore := h.deps.SessionManager.GetStore(projectPath)

	// Special handling for architect sessions
	if req.TicketID == session.ArchitectSessionKey {
		sess, _ := sessStore.GetArchitect()
		if sess == nil {
			writeError(w, http.StatusBadRequest, "no_active_session", "architect does not have an active session")
			return
		}

		if err := sessStore.UpdateStatus(session.ArchitectSessionKey, agentStatus, req.Tool, req.Work); err != nil {
			writeError(w, http.StatusInternalServerError, "update_error", err.Error())
			return
		}

		h.deps.Bus.Emit(events.Event{
			Type:        events.SessionStatus,
			ProjectPath: projectPath,
			TicketID:    req.TicketID,
			Payload: map[string]any{
				"status": req.Status,
				"tool":   req.Tool,
				"work":   req.Work,
			},
		})

		w.WriteHeader(http.StatusOK)
		return
	}

	// Verify ticket exists
	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	_, _, err = store.Get(req.TicketID)
	if err != nil {
		if ticket.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "ticket_not_found", "ticket not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	// Check for active session via session manager
	shortID := storage.ShortID(req.TicketID)
	sess, _ := sessStore.Get(shortID)
	if sess == nil {
		writeError(w, http.StatusBadRequest, "no_active_session", "ticket does not have an active session")
		return
	}

	// Update the session status
	if err := sessStore.UpdateStatus(shortID, agentStatus, req.Tool, req.Work); err != nil {
		writeError(w, http.StatusInternalServerError, "update_error", err.Error())
		return
	}

	h.deps.Bus.Emit(events.Event{
		Type:        events.SessionStatus,
		ProjectPath: projectPath,
		TicketID:    req.TicketID,
		Payload: map[string]any{
			"status": req.Status,
			"tool":   req.Tool,
			"work":   req.Work,
		},
	})

	w.WriteHeader(http.StatusOK)
}

// validAgentStatus checks if the status is a known agent status.
func validAgentStatus(status session.AgentStatus) bool {
	switch status {
	case session.AgentStatusStarting,
		session.AgentStatusInProgress,
		session.AgentStatusIdle,
		session.AgentStatusWaitingPermission,
		session.AgentStatusError:
		return true
	default:
		return false
	}
}
