package api

import (
	"encoding/json"
	"net/http"

	"github.com/kareemaly/cortex/internal/events"
	"github.com/kareemaly/cortex/internal/session"
)

// internalAgentStatuses is the set of statuses that may not be set via
// POST /agent/status. `ended` is produced by the liveness watcher via
// DELETE /sessions/{id} — accepting it here would bypass the clean
// session teardown that EndBySessionID + SessionEnded event provides.
var internalAgentStatuses = map[session.AgentStatus]struct{}{
	session.AgentStatusEnded: {},
}

// AgentHandlers provides HTTP handlers for agent operations.
type AgentHandlers struct {
	deps *Dependencies
}

// NewAgentHandlers creates a new AgentHandlers with the given dependencies.
func NewAgentHandlers(deps *Dependencies) *AgentHandlers {
	return &AgentHandlers{deps: deps}
}

// UpdateAgentStatusRequest is the request body for updating agent status.
// SessionID is the only routing key: it is the canonical UUID minted at
// session creation time.
type UpdateAgentStatusRequest struct {
	SessionID string  `json:"session_id"`
	Status    string  `json:"status"`
	Tool      *string `json:"tool,omitempty"`
	Work      *string `json:"work,omitempty"`
}

// UpdateStatus handles POST /agent/status - updates the agent's current status.
func (h *AgentHandlers) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	var req UpdateAgentStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	if req.SessionID == "" {
		writeError(w, http.StatusBadRequest, "missing_session_id", "session_id is required")
		return
	}

	if req.Status == "" {
		writeError(w, http.StatusBadRequest, "missing_status", "status is required")
		return
	}

	agentStatus := session.AgentStatus(req.Status)
	if !validAgentStatus(agentStatus) {
		writeError(w, http.StatusBadRequest, "invalid_status", "invalid agent status")
		return
	}
	if _, reserved := internalAgentStatuses[agentStatus]; reserved {
		writeError(w, http.StatusBadRequest, "reserved_status",
			"status is internal-only and may not be set via POST")
		return
	}

	if h.deps.SessionManager == nil {
		writeError(w, http.StatusServiceUnavailable, "sessions_unavailable",
			"session manager is not configured")
		return
	}
	projectPath := GetArchitectPath(r.Context())
	sessStore := h.deps.SessionManager.GetStore(projectPath)

	sess, err := sessStore.GetBySessionID(req.SessionID)
	if err != nil || sess == nil {
		writeError(w, http.StatusNotFound, "no_active_session", "no session matches the given session_id")
		return
	}

	if err := sessStore.UpdateStatusBySessionID(sess.SessionID, agentStatus, req.Tool, req.Work); err != nil {
		writeError(w, http.StatusInternalServerError, "update_error", err.Error())
		return
	}

	// Emit sess.TicketID verbatim: ticket sessions get their real ID,
	// architect and collab sessions get an empty string. The store sentinel
	// `ArchitectSessionKey` is intentionally NOT echoed into the event bus —
	// routing by SessionID is the canonical path.
	h.deps.Bus.Emit(events.Event{
		Type:          events.SessionStatus,
		ArchitectPath: projectPath,
		TicketID:      sess.TicketID,
		SessionID:     sess.SessionID,
		Payload: map[string]any{
			"status":     string(agentStatus),
			"tool":       req.Tool,
			"work":       req.Work,
			"session_id": sess.SessionID,
		},
	})

	w.WriteHeader(http.StatusOK)
}

// DebugStatusResponse is what GET /agent/status/debug returns: session and
// supervision counts.
type DebugStatusResponse struct {
	SupervisedCount int `json:"supervised_count"`
}

// DebugStatus handles GET /agent/status/debug. Intentionally global
// (no architect scope) — returns counts across all architects.
func (h *AgentHandlers) DebugStatus(w http.ResponseWriter, r *http.Request) {
	resp := DebugStatusResponse{}
	if h.deps.SessionManager != nil {
		resp.SupervisedCount = h.deps.SessionManager.TotalSessionCount()
	}
	writeJSON(w, http.StatusOK, resp)
}

// validAgentStatus checks if the status is a known agent status.
func validAgentStatus(status session.AgentStatus) bool {
	switch status {
	case session.AgentStatusStarting,
		session.AgentStatusWorking,
		session.AgentStatusIdle,
		session.AgentStatusAwaitingInput,
		session.AgentStatusError,
		session.AgentStatusEnded:
		return true
	default:
		return false
	}
}
