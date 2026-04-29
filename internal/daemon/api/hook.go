package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kareemaly/agentstatus"
)

const maxHookBodyBytes = 1 << 20 // 1 MiB

// HookHandlers handles POST /hook/{agent}.
type HookHandlers struct {
	deps *Dependencies
}

// NewHookHandlers creates a new HookHandlers.
func NewHookHandlers(deps *Dependencies) *HookHandlers {
	return &HookHandlers{deps: deps}
}

// IngestHook handles POST /hook/{agent}.
// This is a GLOBAL route (no X-Cortex-Architect header required) because
// agent CLIs post hooks without project context.
func (h *HookHandlers) IngestHook(w http.ResponseWriter, r *http.Request) {
	if h.deps.HubManager == nil {
		writeError(w, http.StatusServiceUnavailable, "hub_unavailable", "hook ingestion not available")
		return
	}

	agentName := chi.URLParam(r, "agent")
	if agentName == "" {
		writeError(w, http.StatusBadRequest, "missing_agent", "agent name required")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxHookBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		writeError(w, http.StatusBadRequest, "read_error", "failed to read body")
		return
	}

	if err := h.deps.HubManager.Ingest(agentstatus.Agent(agentName), body); err != nil {
		if errors.Is(err, agentstatus.ErrUnknownAgent) {
			writeError(w, http.StatusBadRequest, "unknown_agent",
				"unknown agent: "+agentName)
			return
		}
		writeError(w, http.StatusUnprocessableEntity, "invalid_payload",
			"invalid hook payload: "+err.Error())
		return
	}

	// For OpenCode: on session.created, back-correlate the agent's native session
	// ID to the cortex session UUID so the Hub cache is queryable via
	// sess.AgentSessionID on the next TUI poll cycle.
	if agentName == "opencode" && h.deps.SessionManager != nil {
		h.tryCorrelateOpenCodeSession(body)
	}

	w.WriteHeader(http.StatusAccepted)
}

// tryCorrelateOpenCodeSession parses a session.created payload and, when both
// session_id and cortex_session_id are present, records the mapping so that
// GetEvent(sess.AgentSessionID) resolves to the correct Hub cache entry.
func (h *HookHandlers) tryCorrelateOpenCodeSession(body []byte) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return
	}
	if payload["hook_event_name"] != "session.created" {
		return
	}
	openCodeSessionID, _ := payload["session_id"].(string)
	cortexSessionID, _ := payload["cortex_session_id"].(string)
	if openCodeSessionID == "" || cortexSessionID == "" {
		return
	}
	if err := h.deps.SessionManager.SetAgentSessionIDBySessionID(cortexSessionID, openCodeSessionID); err != nil {
		if h.deps.Logger != nil {
			h.deps.Logger.Debug("opencode session correlation skipped",
				"cortex_session_id", cortexSessionID,
				"opencode_session_id", openCodeSessionID,
				"err", err)
		}
		return
	}
	h.deps.HubManager.RegisterOpenCodeSession(openCodeSessionID, cortexSessionID)
	if h.deps.Logger != nil {
		h.deps.Logger.Info("opencode session correlated",
			"cortex_session_id", cortexSessionID,
			"opencode_session_id", openCodeSessionID)
	}
}
