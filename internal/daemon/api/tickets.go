package api

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/kareemaly/cortex/internal/core/spawn"
	"github.com/kareemaly/cortex/internal/ticket"
	"github.com/kareemaly/cortex/internal/types"
)

// TicketHandlers provides HTTP handlers for ticket operations.
type TicketHandlers struct {
	deps *Dependencies
}

// NewTicketHandlers creates a new TicketHandlers with the given dependencies.
func NewTicketHandlers(deps *Dependencies) *TicketHandlers {
	return &TicketHandlers{deps: deps}
}

// ListAll handles GET /tickets - lists all tickets grouped by status.
func (h *TicketHandlers) ListAll(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())
	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	all, err := store.ListAll()
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	// Apply query filter if specified
	query := strings.ToLower(r.URL.Query().Get("query"))

	resp := ListAllTicketsResponse{
		Backlog:  filterSummaryList(all[ticket.StatusBacklog], ticket.StatusBacklog, query),
		Progress: filterSummaryList(all[ticket.StatusProgress], ticket.StatusProgress, query),
		Review:   filterSummaryList(all[ticket.StatusReview], ticket.StatusReview, query),
		Done:     filterSummaryList(all[ticket.StatusDone], ticket.StatusDone, query),
	}

	// Sort by Updated descending (most recent first)
	sortByUpdated := func(a, b TicketSummary) int {
		return b.Updated.Compare(a.Updated)
	}
	slices.SortFunc(resp.Backlog, sortByUpdated)
	slices.SortFunc(resp.Progress, sortByUpdated)
	slices.SortFunc(resp.Review, sortByUpdated)
	slices.SortFunc(resp.Done, sortByUpdated)

	writeJSON(w, http.StatusOK, resp)
}

// ListByStatus handles GET /tickets/{status} - lists tickets with a specific status.
func (h *TicketHandlers) ListByStatus(w http.ResponseWriter, r *http.Request) {
	status := chi.URLParam(r, "status")
	if !validStatus(status) {
		writeError(w, http.StatusBadRequest, "invalid_status", "invalid status: must be backlog, progress, or done")
		return
	}

	projectPath := GetProjectPath(r.Context())
	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	tickets, err := store.List(ticket.Status(status))
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	// Apply query filter if specified
	query := strings.ToLower(r.URL.Query().Get("query"))

	resp := ListTicketsResponse{
		Tickets: filterSummaryList(tickets, ticket.Status(status), query),
	}

	// Sort by Updated descending (most recent first)
	slices.SortFunc(resp.Tickets, func(a, b TicketSummary) int {
		return b.Updated.Compare(a.Updated)
	})

	writeJSON(w, http.StatusOK, resp)
}

// Create handles POST /tickets - creates a new ticket.
func (h *TicketHandlers) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	projectPath := GetProjectPath(r.Context())
	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	t, err := store.Create(req.Title, req.Body)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	resp := types.ToTicketResponse(t, ticket.StatusBacklog)
	writeJSON(w, http.StatusCreated, resp)
}

// Get handles GET /tickets/{status}/{id} - gets a specific ticket.
func (h *TicketHandlers) Get(w http.ResponseWriter, r *http.Request) {
	status := chi.URLParam(r, "status")
	if !validStatus(status) {
		writeError(w, http.StatusBadRequest, "invalid_status", "invalid status: must be backlog, progress, or done")
		return
	}

	projectPath := GetProjectPath(r.Context())
	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	id := chi.URLParam(r, "id")
	t, actualStatus, err := store.Get(id)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	// Verify the ticket is in the expected status
	if string(actualStatus) != status {
		writeError(w, http.StatusNotFound, "not_found", "ticket not found in specified status")
		return
	}

	resp := types.ToTicketResponse(t, actualStatus)
	writeJSON(w, http.StatusOK, resp)
}

// Update handles PUT /tickets/{status}/{id} - updates a ticket.
func (h *TicketHandlers) Update(w http.ResponseWriter, r *http.Request) {
	status := chi.URLParam(r, "status")
	if !validStatus(status) {
		writeError(w, http.StatusBadRequest, "invalid_status", "invalid status: must be backlog, progress, or done")
		return
	}

	projectPath := GetProjectPath(r.Context())
	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	id := chi.URLParam(r, "id")

	// Check ticket exists and is in the expected status
	_, actualStatus, err := store.Get(id)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}
	if string(actualStatus) != status {
		writeError(w, http.StatusNotFound, "not_found", "ticket not found in specified status")
		return
	}

	var req UpdateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	t, err := store.Update(id, req.Title, req.Body)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	resp := types.ToTicketResponse(t, actualStatus)
	writeJSON(w, http.StatusOK, resp)
}

// Delete handles DELETE /tickets/{status}/{id} - deletes a ticket.
func (h *TicketHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	status := chi.URLParam(r, "status")
	if !validStatus(status) {
		writeError(w, http.StatusBadRequest, "invalid_status", "invalid status: must be backlog, progress, or done")
		return
	}

	projectPath := GetProjectPath(r.Context())
	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	id := chi.URLParam(r, "id")

	// Check ticket exists and is in the expected status
	_, actualStatus, err := store.Get(id)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}
	if string(actualStatus) != status {
		writeError(w, http.StatusNotFound, "not_found", "ticket not found in specified status")
		return
	}

	if err := store.Delete(id); err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Move handles POST /tickets/{status}/{id}/move - moves a ticket to a different status.
func (h *TicketHandlers) Move(w http.ResponseWriter, r *http.Request) {
	status := chi.URLParam(r, "status")
	if !validStatus(status) {
		writeError(w, http.StatusBadRequest, "invalid_status", "invalid status: must be backlog, progress, or done")
		return
	}

	projectPath := GetProjectPath(r.Context())
	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	id := chi.URLParam(r, "id")

	// Check ticket exists and is in the expected status
	_, actualStatus, err := store.Get(id)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}
	if string(actualStatus) != status {
		writeError(w, http.StatusNotFound, "not_found", "ticket not found in specified status")
		return
	}

	var req MoveTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	if !validStatus(req.To) {
		writeError(w, http.StatusBadRequest, "invalid_status", "invalid target status: must be backlog, progress, or done")
		return
	}

	if err := store.Move(id, ticket.Status(req.To)); err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	// Fetch the updated ticket
	t, newStatus, err := store.Get(id)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	resp := types.ToTicketResponse(t, newStatus)
	writeJSON(w, http.StatusOK, resp)
}

// Spawn handles POST /tickets/{status}/{id}/spawn - spawns a session.
func (h *TicketHandlers) Spawn(w http.ResponseWriter, r *http.Request) {
	status := chi.URLParam(r, "status")
	if !validStatus(status) {
		writeError(w, http.StatusBadRequest, "invalid_status", "invalid status: must be backlog, progress, or done")
		return
	}

	id := chi.URLParam(r, "id")
	mode := r.URL.Query().Get("mode")
	projectPath := GetProjectPath(r.Context())

	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	// Verify ticket exists at expected status (HTTP-specific URL validation)
	_, actualStatus, err := store.Get(id)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}
	if string(actualStatus) != status {
		writeError(w, http.StatusNotFound, "not_found", "ticket not found in specified status")
		return
	}

	// Check tmux is available
	if h.deps.TmuxManager == nil {
		writeError(w, http.StatusServiceUnavailable, "tmux_unavailable", "tmux is not installed")
		return
	}

	// Delegate to shared orchestration
	result, err := spawn.Orchestrate(r.Context(), spawn.OrchestrateRequest{
		TicketID:    id,
		Mode:        mode,
		ProjectPath: projectPath,
	}, spawn.OrchestrateDeps{
		Store:       store,
		TmuxManager: h.deps.TmuxManager,
		Logger:      h.deps.Logger,
	})
	if err != nil {
		switch {
		case spawn.IsStateError(err):
			writeError(w, http.StatusConflict, "state_conflict", err.Error())
		case spawn.IsConfigError(err):
			writeError(w, http.StatusBadRequest, "config_error", err.Error())
		case spawn.IsBinaryNotFoundError(err):
			h.deps.Logger.Error("binary not found", "error", err)
			writeError(w, http.StatusInternalServerError, "spawn_error", err.Error())
		default:
			handleTicketError(w, err, h.deps.Logger)
		}
		return
	}

	// Already active: focus window and return existing session
	if result.Outcome == spawn.OutcomeAlreadyActive {
		if err := h.deps.TmuxManager.FocusWindow(result.TmuxSession, result.StateInfo.Session.TmuxWindow); err != nil {
			h.deps.Logger.Warn("failed to focus window", "error", err)
		}
		resp := SpawnResponse{
			Session: types.ToSessionResponse(*result.StateInfo.Session),
			Ticket:  types.ToTicketResponse(result.Ticket, result.TicketStatus),
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}

	// Spawned or resumed
	resp := SpawnResponse{
		Session: types.ToSessionResponse(*result.Ticket.Session),
		Ticket:  types.ToTicketResponse(result.Ticket, result.TicketStatus),
	}
	writeJSON(w, http.StatusCreated, resp)
}
