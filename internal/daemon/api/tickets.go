package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kareemaly/cortex/internal/core/spawn"
	"github.com/kareemaly/cortex/internal/events"
	projectconfig "github.com/kareemaly/cortex/internal/project/config"
	"github.com/kareemaly/cortex/internal/storage"
	"github.com/kareemaly/cortex/internal/ticket"
	"github.com/kareemaly/cortex/internal/tmux"
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

	// Parse due_before filter if specified (RFC3339 format)
	var dueBefore *time.Time
	if dueBeforeStr := r.URL.Query().Get("due_before"); dueBeforeStr != "" {
		parsed, err := time.Parse(time.RFC3339, dueBeforeStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_due_before", "due_before must be in RFC3339 format")
			return
		}
		dueBefore = &parsed
	}

	// Load project config to get tmux session name for orphan detection.
	tmuxSession := ""
	projectCfg, cfgErr := projectconfig.Load(projectPath)
	if cfgErr == nil && projectCfg.Name != "" {
		tmuxSession = projectCfg.Name
	}

	resp := ListAllTicketsResponse{
		Backlog:  filterSummaryList(all[ticket.StatusBacklog], ticket.StatusBacklog, query, dueBefore, tmuxSession, h.deps.TmuxManager, h.deps.SessionManager, projectPath),
		Progress: filterSummaryList(all[ticket.StatusProgress], ticket.StatusProgress, query, dueBefore, tmuxSession, h.deps.TmuxManager, h.deps.SessionManager, projectPath),
		Done:     filterSummaryList(all[ticket.StatusDone], ticket.StatusDone, query, dueBefore, tmuxSession, h.deps.TmuxManager, h.deps.SessionManager, projectPath),
	}

	// Sort by Created descending (most recent first)
	sortByCreated := func(a, b TicketSummary) int {
		return b.Created.Compare(a.Created)
	}
	slices.SortFunc(resp.Backlog, sortByCreated)
	slices.SortFunc(resp.Progress, sortByCreated)
	slices.SortFunc(resp.Done, sortByCreated)

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

	// Parse due_before filter if specified (RFC3339 format)
	var dueBefore *time.Time
	if dueBeforeStr := r.URL.Query().Get("due_before"); dueBeforeStr != "" {
		parsed, err := time.Parse(time.RFC3339, dueBeforeStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_due_before", "due_before must be in RFC3339 format")
			return
		}
		dueBefore = &parsed
	}

	// Load project config to get tmux session name for orphan detection.
	tmuxSession := ""
	projectCfg, cfgErr := projectconfig.Load(projectPath)
	if cfgErr == nil && projectCfg.Name != "" {
		tmuxSession = projectCfg.Name
	}

	resp := ListTicketsResponse{
		Tickets: filterSummaryList(tickets, ticket.Status(status), query, dueBefore, tmuxSession, h.deps.TmuxManager, h.deps.SessionManager, projectPath),
	}

	// Sort by Created descending (most recent first)
	slices.SortFunc(resp.Tickets, func(a, b TicketSummary) int {
		return b.Created.Compare(a.Created)
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

	// Validate ticket type
	ticketType := req.Type
	if ticketType == "" {
		ticketType = "work"
	}
	if ticketType != "work" && ticketType != "research" {
		writeError(w, http.StatusBadRequest, "invalid_type",
			fmt.Sprintf("invalid ticket type %q, valid types: work, research", ticketType))
		return
	}

	// Parse due date if provided (RFC3339 format)
	var dueDate *time.Time
	if req.DueDate != nil && *req.DueDate != "" {
		parsed, err := time.Parse(time.RFC3339, *req.DueDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_due_date", "due_date must be in RFC3339 format")
			return
		}
		dueDate = &parsed
	}

	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	t, err := store.Create(req.Title, req.Body, ticketType, dueDate, req.References, req.Repo)
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

	t, err := store.Update(id, req.Title, req.Body, req.References)
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

// SetDueDate handles PATCH /tickets/{id}/due-date - sets the due date for a ticket.
func (h *TicketHandlers) SetDueDate(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())
	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	id := chi.URLParam(r, "id")

	var req SetDueDateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	if req.DueDate == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "due_date is required")
		return
	}

	// Parse due date (RFC3339 format)
	dueDate, err := time.Parse(time.RFC3339, req.DueDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_due_date", "due_date must be in RFC3339 format")
		return
	}

	t, err := store.SetDueDate(id, &dueDate)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	// Get status for response
	_, status, err := store.Get(id)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	resp := types.ToTicketResponse(t, status)
	writeJSON(w, http.StatusOK, resp)
}

// ClearDueDate handles DELETE /tickets/{id}/due-date - removes the due date from a ticket.
func (h *TicketHandlers) ClearDueDate(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())
	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	id := chi.URLParam(r, "id")

	t, err := store.ClearDueDate(id)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	// Get status for response
	_, status, err := store.Get(id)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	resp := types.ToTicketResponse(t, status)
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
	sessionStore := h.deps.SessionManager.GetStore(projectPath)
	result, err := spawn.Orchestrate(r.Context(), spawn.OrchestrateRequest{
		TicketID:    id,
		Mode:        mode,
		ProjectPath: projectPath,
	}, spawn.OrchestrateDeps{
		Store:        store,
		SessionStore: sessionStore,
		TmuxManager:  h.deps.TmuxManager,
		Logger:       h.deps.Logger,
		CortexdPath:  h.deps.CortexdPath,
		DefaultsDir:  h.deps.DefaultsDir,
	})
	if err != nil {
		switch {
		case spawn.IsStateError(err):
			stateErr := err.(*spawn.StateError)
			if stateErr.State == spawn.StateOrphaned {
				writeError(w, http.StatusConflict, "session_orphaned", err.Error())
			} else {
				writeError(w, http.StatusConflict, "state_conflict", err.Error())
			}
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
		if result.StateInfo.Session != nil {
			if err := h.deps.TmuxManager.FocusWindow(result.TmuxSession, result.StateInfo.Session.TmuxWindow); err != nil {
				h.deps.Logger.Warn("failed to focus window", "error", err)
			}
			if err := h.deps.TmuxManager.SwitchClient(result.TmuxSession); err != nil {
				h.deps.Logger.Warn("failed to switch tmux client", "session", result.TmuxSession, "error", err)
			}
		}
		resp := SpawnResponse{
			Session: types.ToSessionResponse(result.StateInfo.Session),
			Ticket:  types.ToTicketResponse(result.Ticket, result.TicketStatus),
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}

	// Spawned or resumed: look up session from session store
	sess, _ := sessionStore.GetByTicketID(id)
	resp := SpawnResponse{
		Ticket: types.ToTicketResponse(result.Ticket, result.TicketStatus),
	}
	if sess != nil {
		resp.Session = types.ToSessionResponse(sess)
	}
	h.deps.Bus.Emit(events.Event{
		Type:        events.SessionStarted,
		ProjectPath: projectPath,
		TicketID:    id,
	})
	writeJSON(w, http.StatusCreated, resp)
}

// GetByID handles GET /tickets/by-id/{id} - gets a ticket by ID regardless of status.
func (h *TicketHandlers) GetByID(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())
	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	id := chi.URLParam(r, "id")
	t, status, err := store.Get(id)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	resp := types.ToTicketResponse(t, status)
	writeJSON(w, http.StatusOK, resp)
}

// Focus handles POST /tickets/{id}/focus - focuses the tmux window of a ticket's active session.
func (h *TicketHandlers) Focus(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())
	id := chi.URLParam(r, "id")

	// Look up session from session manager
	if h.deps.SessionManager == nil {
		writeError(w, http.StatusNotFound, "no_active_session", "no session manager available")
		return
	}

	sessStore := h.deps.SessionManager.GetStore(projectPath)
	sess, err := sessStore.GetByTicketID(id)
	if err != nil || sess == nil || sess.TmuxWindow == "" {
		writeError(w, http.StatusNotFound, "no_active_session", "ticket has no active session with a tmux window")
		return
	}

	if h.deps.TmuxManager == nil {
		writeError(w, http.StatusServiceUnavailable, "tmux_unavailable", "tmux is not installed")
		return
	}

	projectCfg, cfgErr := projectconfig.Load(projectPath)
	tmuxSession := "cortex"
	if cfgErr == nil && projectCfg.Name != "" {
		tmuxSession = projectCfg.Name
	}

	if err := h.deps.TmuxManager.FocusWindow(tmuxSession, sess.TmuxWindow); err != nil {
		writeError(w, http.StatusInternalServerError, "focus_error", err.Error())
		return
	}

	if err := h.deps.TmuxManager.SwitchClient(tmuxSession); err != nil {
		h.deps.Logger.Warn("failed to switch tmux client", "session", tmuxSession, "error", err)
	}

	resp := FocusResponse{
		Success: true,
		Window:  sess.TmuxWindow,
	}
	writeJSON(w, http.StatusOK, resp)
}

// Conclude handles POST /tickets/{id}/conclude - concludes a session and moves ticket to done.
func (h *TicketHandlers) Conclude(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())
	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	id := chi.URLParam(r, "id")
	t, _, err := store.Get(id)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	var req ConcludeSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "content cannot be empty")
		return
	}

	// Capture session info before ending
	var tmuxWindow string

	if h.deps.SessionManager != nil {
		sessStore := h.deps.SessionManager.GetStore(projectPath)
		if sess, sessErr := sessStore.GetByTicketID(id); sessErr == nil && sess != nil {
			tmuxWindow = sess.TmuxWindow
		}
	}

	// End the ephemeral session
	if h.deps.SessionManager != nil {
		sessStore := h.deps.SessionManager.GetStore(projectPath)
		shortID := storage.ShortID(id)
		if endErr := sessStore.End(shortID); endErr != nil {
			h.deps.Logger.Warn("failed to end session", "error", endErr)
		}
	}

	h.deps.Bus.Emit(events.Event{
		Type:        events.SessionEnded,
		ProjectPath: projectPath,
		TicketID:    id,
	})

	// Create conclusion record
	conclusionType := req.Type
	if conclusionType == "" {
		conclusionType = t.Type
	}
	repo := req.Repo
	if repo == "" {
		repo = t.Repo
	}

	var conclusionID string
	if h.deps.ConclusionStoreManager != nil {
		conclusionStore, csErr := h.deps.ConclusionStoreManager.GetStore(projectPath)
		if csErr == nil {
			conclusion, createErr := conclusionStore.Create(conclusionType, id, repo, req.Content)
			if createErr != nil {
				h.deps.Logger.Warn("failed to create conclusion", "error", createErr)
			} else {
				conclusionID = conclusion.ID
			}
		}
	}

	// Update ticket with session back-reference
	if conclusionID != "" {
		if _, setErr := store.SetSession(id, conclusionID); setErr != nil {
			h.deps.Logger.Warn("failed to set session back-reference", "error", setErr)
		}
	}

	// Move the ticket to done
	if err := store.Move(id, ticket.StatusDone); err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	// Kill tmux window if associated (best-effort)
	if tmuxWindow != "" && h.deps.TmuxManager != nil {
		projectCfg, cfgErr := projectconfig.Load(projectPath)
		tmuxSession := "cortex"
		if cfgErr == nil && projectCfg.Name != "" {
			tmuxSession = projectCfg.Name
		}

		if killErr := h.deps.TmuxManager.KillWindow(tmuxSession, tmuxWindow); killErr != nil {
			if !tmux.IsWindowNotFound(killErr) && !tmux.IsSessionNotFound(killErr) {
				h.deps.Logger.Warn("failed to kill tmux window", "window", tmuxWindow, "error", killErr)
			}
		}
	}

	resp := ConcludeSessionResponse{
		Success:  true,
		TicketID: id,
		Message:  "Session concluded and ticket moved to done",
	}
	writeJSON(w, http.StatusOK, resp)
}

// Edit handles POST /tickets/{id}/edit - opens the ticket's index.md in $EDITOR via tmux popup.
func (h *TicketHandlers) Edit(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())
	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	ticketID := chi.URLParam(r, "id")

	// Get the file path for the ticket's index.md
	indexPath, err := store.IndexPath(ticketID)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	// Check tmux is available
	if h.deps.TmuxManager == nil {
		writeError(w, http.StatusServiceUnavailable, "tmux_unavailable", "tmux is not installed")
		return
	}

	// Build editor command
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	command := fmt.Sprintf("%s %q", editor, indexPath)

	// Get tmux session name
	projectCfg, cfgErr := projectconfig.Load(projectPath)
	tmuxSession := "cortex"
	if cfgErr == nil && projectCfg.Name != "" {
		tmuxSession = projectCfg.Name
	}

	// Execute popup
	if err := h.deps.TmuxManager.DisplayPopup(tmuxSession, "", command); err != nil {
		writeError(w, http.StatusInternalServerError, "tmux_error", fmt.Sprintf("failed to display popup: %s", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, ExecuteActionResponse{
		Success: true,
		Message: "Editor opened",
	})
}

// ShowPopup handles POST /tickets/{id}/show-popup - opens cortex ticket in a tmux popup.
func (h *TicketHandlers) ShowPopup(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())
	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	ticketID := chi.URLParam(r, "id")

	// Validate ticket exists
	_, _, err = store.Get(ticketID)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	// Check tmux is available
	if h.deps.TmuxManager == nil {
		writeError(w, http.StatusServiceUnavailable, "tmux_unavailable", "tmux is not installed")
		return
	}

	// Build command
	command := fmt.Sprintf("cortex ticket %s", ticketID)

	// Get tmux session name
	projectCfg, cfgErr := projectconfig.Load(projectPath)
	tmuxSession := "cortex"
	if cfgErr == nil && projectCfg.Name != "" {
		tmuxSession = projectCfg.Name
	}

	// Execute popup
	if err := h.deps.TmuxManager.DisplayPopup(tmuxSession, projectPath, command); err != nil {
		writeError(w, http.StatusInternalServerError, "tmux_error", fmt.Sprintf("failed to display popup: %s", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, ExecuteActionResponse{
		Success: true,
		Message: "Popup opened",
	})
}
