package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/kareemaly/cortex/internal/core/spawn"
	projectconfig "github.com/kareemaly/cortex/internal/project/config"
	"github.com/kareemaly/cortex/internal/ticket"
	"github.com/kareemaly/cortex/internal/tmux"
	"github.com/kareemaly/cortex/internal/types"
	"github.com/kareemaly/cortex/internal/worktree"
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

// AddComment handles POST /tickets/{id}/comments - adds a comment to a ticket.
func (h *TicketHandlers) AddComment(w http.ResponseWriter, r *http.Request) {
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

	var req AddCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	// Validate comment type
	commentType := ticket.CommentType(req.Type)
	switch commentType {
	case ticket.CommentScopeChange, ticket.CommentDecision, ticket.CommentBlocker,
		ticket.CommentProgress, ticket.CommentQuestion, ticket.CommentRejection,
		ticket.CommentGeneral, ticket.CommentTicketDone:
		// Valid type
	default:
		writeError(w, http.StatusBadRequest, "validation_error", "invalid comment type")
		return
	}

	// Find active session ID
	var activeSessionID string
	if t.Session != nil && t.Session.IsActive() {
		activeSessionID = t.Session.ID
	}

	comment, err := store.AddComment(id, activeSessionID, commentType, req.Content)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	resp := AddCommentResponse{
		Success: true,
		Comment: types.ToCommentResponse(comment),
	}
	writeJSON(w, http.StatusOK, resp)
}

// RequestReview handles POST /tickets/{id}/reviews - requests a review for a ticket.
func (h *TicketHandlers) RequestReview(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())
	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	id := chi.URLParam(r, "id")

	var req RequestReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	if req.RepoPath == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "repo_path cannot be empty")
		return
	}
	if req.Summary == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "summary cannot be empty")
		return
	}

	reviewCount, err := store.AddReviewRequest(id, req.RepoPath, req.Summary)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	// Move ticket to review status (idempotent - no-op if already in review or done)
	_, currentStatus, err := store.Get(id)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}
	if currentStatus != ticket.StatusReview && currentStatus != ticket.StatusDone {
		if err := store.Move(id, ticket.StatusReview); err != nil {
			handleTicketError(w, err, h.deps.Logger)
			return
		}
	}

	resp := RequestReviewResponse{
		Success:     true,
		Message:     "Review request added. Wait for human approval.",
		ReviewCount: reviewCount,
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

	if req.FullReport == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "full_report cannot be empty")
		return
	}

	// Capture session info before ending
	var worktreePath, featureBranch *string
	var activeSessionID string
	var tmuxWindow string
	if t.Session != nil {
		worktreePath = t.Session.WorktreePath
		featureBranch = t.Session.FeatureBranch
		tmuxWindow = t.Session.TmuxWindow
		if t.Session.IsActive() {
			activeSessionID = t.Session.ID
		}
	}

	// Add ticket_done comment with the full report
	_, err = store.AddComment(id, activeSessionID, ticket.CommentTicketDone, req.FullReport)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	// End the session
	if err := store.EndSession(id); err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	// Move the ticket to done
	if err := store.Move(id, ticket.StatusDone); err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	// Cleanup worktree if present (best-effort)
	if worktreePath != nil && featureBranch != nil && projectPath != "" {
		wm := worktree.NewManager(projectPath)
		if err := wm.Remove(context.Background(), *worktreePath, *featureBranch); err != nil {
			log.Printf("warning: failed to cleanup worktree: %v", err)
		}
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
				log.Printf("warning: failed to kill tmux window %q: %v", tmuxWindow, killErr)
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
