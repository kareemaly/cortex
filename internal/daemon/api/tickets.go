package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/kareemaly/cortex/internal/core/spawn"
	projectconfig "github.com/kareemaly/cortex/internal/project/config"
	"github.com/kareemaly/cortex/internal/ticket"
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
		handleTicketError(w, err)
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
		handleTicketError(w, err)
		return
	}

	// Apply query filter if specified
	query := strings.ToLower(r.URL.Query().Get("query"))

	resp := ListTicketsResponse{
		Tickets: filterSummaryList(tickets, ticket.Status(status), query),
	}

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
		handleTicketError(w, err)
		return
	}

	resp := toTicketResponse(t, ticket.StatusBacklog)
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
		handleTicketError(w, err)
		return
	}

	// Verify the ticket is in the expected status
	if string(actualStatus) != status {
		writeError(w, http.StatusNotFound, "not_found", "ticket not found in specified status")
		return
	}

	resp := toTicketResponse(t, actualStatus)
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
		handleTicketError(w, err)
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
		handleTicketError(w, err)
		return
	}

	resp := toTicketResponse(t, actualStatus)
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
		handleTicketError(w, err)
		return
	}
	if string(actualStatus) != status {
		writeError(w, http.StatusNotFound, "not_found", "ticket not found in specified status")
		return
	}

	if err := store.Delete(id); err != nil {
		handleTicketError(w, err)
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
		handleTicketError(w, err)
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
		handleTicketError(w, err)
		return
	}

	// Fetch the updated ticket
	t, newStatus, err := store.Get(id)
	if err != nil {
		handleTicketError(w, err)
		return
	}

	resp := toTicketResponse(t, newStatus)
	writeJSON(w, http.StatusOK, resp)
}

// Spawn handles POST /tickets/{status}/{id}/spawn - spawns a session.
func (h *TicketHandlers) Spawn(w http.ResponseWriter, r *http.Request) {
	status := chi.URLParam(r, "status")
	if !validStatus(status) {
		writeError(w, http.StatusBadRequest, "invalid_status", "invalid status: must be backlog, progress, or done")
		return
	}

	// Parse mode parameter
	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "normal"
	}
	if mode != "normal" && mode != "resume" && mode != "fresh" {
		writeError(w, http.StatusBadRequest, "invalid_mode", "mode must be normal, resume, or fresh")
		return
	}

	projectPath := GetProjectPath(r.Context())
	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	// Load project config for this specific project
	projectCfg, err := projectconfig.Load(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "config_error", "failed to load project config")
		return
	}

	id := chi.URLParam(r, "id")

	// Get ticket and verify status
	t, actualStatus, err := store.Get(id)
	if err != nil {
		handleTicketError(w, err)
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

	// Get session name
	sessionName := projectCfg.Name
	if sessionName == "" {
		sessionName = "cortex"
	}

	// Detect session state
	stateInfo, err := spawn.DetectTicketState(t, sessionName, h.deps.TmuxManager)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "state_error", "failed to detect session state")
		return
	}

	// Apply mode/state matrix
	switch mode {
	case "normal":
		switch stateInfo.State {
		case spawn.StateNormal, spawn.StateEnded:
			// Spawn new - continue
		case spawn.StateActive:
			// Focus window and return existing
			if err := h.deps.TmuxManager.FocusWindow(sessionName, stateInfo.Session.TmuxWindow); err != nil {
				h.deps.Logger.Warn("failed to focus window", "error", err)
			}
			resp := SpawnResponse{
				Session: toSessionResponse(*stateInfo.Session),
				Ticket:  toTicketResponse(t, actualStatus),
			}
			writeJSON(w, http.StatusOK, resp)
			return
		case spawn.StateOrphaned:
			writeError(w, http.StatusConflict, "session_orphaned", "ticket has an orphaned session; use mode=resume or mode=fresh")
			return
		}

	case "resume":
		switch stateInfo.State {
		case spawn.StateOrphaned:
			// Resume - handled below
		case spawn.StateNormal:
			writeError(w, http.StatusBadRequest, "no_session_to_resume", "no session to resume")
			return
		case spawn.StateActive:
			writeError(w, http.StatusConflict, "session_active", "ticket already has an active session")
			return
		case spawn.StateEnded:
			writeError(w, http.StatusBadRequest, "session_ended", "session has ended; cannot resume")
			return
		}

	case "fresh":
		switch stateInfo.State {
		case spawn.StateOrphaned, spawn.StateEnded:
			// Clear and spawn new - handled below
		case spawn.StateNormal:
			writeError(w, http.StatusBadRequest, "no_session_to_clear", "no session to clear")
			return
		case spawn.StateActive:
			writeError(w, http.StatusConflict, "session_active", "ticket has an active session; cannot clear")
			return
		}
	}

	// Create spawner
	ticketsDir := filepath.Join(projectPath, ".cortex", "tickets")
	spawner := spawn.NewSpawner(spawn.Dependencies{
		Store:       store,
		TmuxManager: h.deps.TmuxManager,
	})

	var result *spawn.SpawnResult

	if mode == "resume" {
		// Resume orphaned session
		result, err = spawner.Resume(spawn.ResumeRequest{
			AgentType:   spawn.AgentTypeTicketAgent,
			TmuxSession: sessionName,
			ProjectPath: projectPath,
			TicketsDir:  ticketsDir,
			SessionID:   stateInfo.Session.ID,
			WindowName:  stateInfo.Session.TmuxWindow,
			TicketID:    id,
		})
	} else {
		// Fresh mode: end existing session first
		if mode == "fresh" && stateInfo.Session != nil {
			if err := store.EndSession(id); err != nil {
				h.deps.Logger.Warn("failed to end session", "error", err)
			}
		}

		// Spawn new
		result, err = spawner.Spawn(spawn.SpawnRequest{
			AgentType:   spawn.AgentTypeTicketAgent,
			Agent:       string(projectCfg.Agent),
			TmuxSession: sessionName,
			ProjectPath: projectPath,
			TicketsDir:  ticketsDir,
			TicketID:    id,
			Ticket:      t,
		})
	}

	if err != nil {
		h.deps.Logger.Error("failed to spawn agent", "error", err)
		writeError(w, http.StatusInternalServerError, "spawn_error", "failed to spawn agent session")
		return
	}

	if !result.Success {
		writeError(w, http.StatusInternalServerError, "spawn_error", result.Message)
		return
	}

	// Move ticket to progress if in backlog
	if actualStatus == ticket.StatusBacklog {
		if err := store.Move(id, ticket.StatusProgress); err != nil {
			h.deps.Logger.Warn("failed to move ticket to progress", "error", err)
		}
	}

	// Fetch updated ticket
	t, newStatus, err := store.Get(id)
	if err != nil {
		handleTicketError(w, err)
		return
	}

	resp := SpawnResponse{
		Session: toSessionResponse(*t.Session),
		Ticket:  toTicketResponse(t, newStatus),
	}
	writeJSON(w, http.StatusCreated, resp)
}
