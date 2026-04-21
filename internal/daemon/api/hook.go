package api

import (
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

	w.WriteHeader(http.StatusAccepted)
}
