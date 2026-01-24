package api

import (
	"encoding/json"
	"net/http"

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
	agentStatus := ticket.AgentStatus(req.Status)
	if !validAgentStatus(agentStatus) {
		writeError(w, http.StatusBadRequest, "invalid_status", "invalid agent status")
		return
	}

	projectPath := GetProjectPath(r.Context())
	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	// Get ticket to verify it exists and has an active session
	t, _, err := store.Get(req.TicketID)
	if err != nil {
		if ticket.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "ticket_not_found", "ticket not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	if t.Session == nil || t.Session.EndedAt != nil {
		writeError(w, http.StatusBadRequest, "no_active_session", "ticket does not have an active session")
		return
	}

	// Update the session status
	if err := store.UpdateSessionStatus(req.TicketID, agentStatus, req.Tool, req.Work); err != nil {
		writeError(w, http.StatusInternalServerError, "update_error", err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

// validAgentStatus checks if the status is a known agent status.
func validAgentStatus(status ticket.AgentStatus) bool {
	switch status {
	case ticket.AgentStatusStarting,
		ticket.AgentStatusInProgress,
		ticket.AgentStatusIdle,
		ticket.AgentStatusWaitingPermission,
		ticket.AgentStatusError:
		return true
	default:
		return false
	}
}
