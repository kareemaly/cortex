package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/kareemaly/cortex/internal/core/spawn"
	daemonconfig "github.com/kareemaly/cortex/internal/daemon/config"
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

	// Sort by Created descending (most recent first)
	sortByCreated := func(a, b TicketSummary) int {
		return b.Created.Compare(a.Created)
	}
	slices.SortFunc(resp.Backlog, sortByCreated)
	slices.SortFunc(resp.Progress, sortByCreated)
	slices.SortFunc(resp.Review, sortByCreated)
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

	resp := ListTicketsResponse{
		Tickets: filterSummaryList(tickets, ticket.Status(status), query),
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
	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	t, err := store.Create(req.Title, req.Body, req.Type)
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
		CortexdPath: h.deps.CortexdPath,
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
		if err := h.deps.TmuxManager.FocusWindow(result.TmuxSession, result.StateInfo.Session.TmuxWindow); err != nil {
			h.deps.Logger.Warn("failed to focus window", "error", err)
		}
		if err := h.deps.TmuxManager.SwitchClient(result.TmuxSession); err != nil {
			h.deps.Logger.Warn("failed to switch tmux client", "session", result.TmuxSession, "error", err)
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
	case ticket.CommentReviewRequested, ticket.CommentDone, ticket.CommentBlocker,
		ticket.CommentGeneral:
		// Valid type
	default:
		writeError(w, http.StatusBadRequest, "validation_error", "invalid comment type: must be review_requested, done, blocker, or comment")
		return
	}

	// Find active session ID
	var activeSessionID string
	if t.Session != nil && t.Session.IsActive() {
		activeSessionID = t.Session.ID
	}

	// Convert action from request
	var action *ticket.CommentAction
	if req.Action != nil {
		action = &ticket.CommentAction{
			Type: req.Action.Type,
			Args: req.Action.Args,
		}
	}

	comment, err := store.AddComment(id, activeSessionID, commentType, req.Content, action)
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
	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "content cannot be empty")
		return
	}

	// Find active session ID
	t, _, err := store.Get(id)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}
	var activeSessionID string
	if t.Session != nil && t.Session.IsActive() {
		activeSessionID = t.Session.ID
	}

	// Build git_diff action
	args := ticket.GitDiffArgs{RepoPath: req.RepoPath}
	if req.Commit != "" {
		args.Commit = req.Commit
	}
	action := &ticket.CommentAction{
		Type: "git_diff",
		Args: args,
	}

	comment, err := store.AddComment(id, activeSessionID, ticket.CommentReviewRequested, req.Content, action)
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
		Success: true,
		Message: "Review request added. Wait for human approval.",
		Comment: types.ToCommentResponse(comment),
	}
	writeJSON(w, http.StatusOK, resp)
}

// Focus handles POST /tickets/{id}/focus - focuses the tmux window of a ticket's active session.
func (h *TicketHandlers) Focus(w http.ResponseWriter, r *http.Request) {
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

	if t.Session == nil || !t.Session.IsActive() || t.Session.TmuxWindow == "" {
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

	if err := h.deps.TmuxManager.FocusWindow(tmuxSession, t.Session.TmuxWindow); err != nil {
		writeError(w, http.StatusInternalServerError, "focus_error", err.Error())
		return
	}

	if err := h.deps.TmuxManager.SwitchClient(tmuxSession); err != nil {
		h.deps.Logger.Warn("failed to switch tmux client", "session", tmuxSession, "error", err)
	}

	resp := FocusResponse{
		Success: true,
		Window:  t.Session.TmuxWindow,
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

	// Add done comment with the report
	_, err = store.AddComment(id, activeSessionID, ticket.CommentDone, req.Content, nil)
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

// ExecuteAction handles POST /tickets/{id}/comments/{comment_id}/execute - executes a comment action.
func (h *TicketHandlers) ExecuteAction(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())
	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	ticketID := chi.URLParam(r, "id")
	commentID := chi.URLParam(r, "comment_id")

	// Get ticket
	t, _, err := store.Get(ticketID)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	// Find comment by ID
	var comment *ticket.Comment
	for i := range t.Comments {
		if t.Comments[i].ID == commentID {
			comment = &t.Comments[i]
			break
		}
	}
	if comment == nil {
		writeError(w, http.StatusNotFound, "comment_not_found", "comment not found")
		return
	}

	// Verify comment has an action
	if comment.Action == nil {
		writeError(w, http.StatusBadRequest, "no_action", "comment has no action")
		return
	}

	// Verify ticket has active session
	if t.Session == nil || !t.Session.IsActive() {
		writeError(w, http.StatusConflict, "no_active_session", "ticket has no active session")
		return
	}

	// Check tmux is available
	if h.deps.TmuxManager == nil {
		writeError(w, http.StatusServiceUnavailable, "tmux_unavailable", "tmux is not installed")
		return
	}

	// Only support git_diff action type
	if comment.Action.Type != "git_diff" {
		writeError(w, http.StatusBadRequest, "unsupported_action", fmt.Sprintf("unsupported action type: %s", comment.Action.Type))
		return
	}

	// Parse git_diff args
	argsMap, ok := comment.Action.Args.(map[string]any)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_args", "invalid action args")
		return
	}
	repoPath, _ := argsMap["repo_path"].(string)
	commit, _ := argsMap["commit"].(string)

	if repoPath == "" {
		writeError(w, http.StatusBadRequest, "invalid_repo_path", "repo_path is required")
		return
	}

	// Validate repo_path exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		writeError(w, http.StatusBadRequest, "invalid_repo_path", fmt.Sprintf("repo_path does not exist: %s", repoPath))
		return
	}

	// Load global config for git_diff_tool setting
	cfg, err := daemonconfig.Load()
	if err != nil {
		h.deps.Logger.Warn("failed to load daemon config, using default", "error", err)
		cfg = daemonconfig.DefaultConfig()
	}

	// Build command based on tool
	var command string
	switch cfg.GitDiffTool {
	case "lazygit":
		command = "lazygit"
	default:
		// Use git diff
		if commit != "" {
			command = fmt.Sprintf("git diff %s", commit)
		} else {
			command = "git diff"
		}
	}

	// Get tmux session name
	projectCfg, cfgErr := projectconfig.Load(projectPath)
	tmuxSession := "cortex"
	if cfgErr == nil && projectCfg.Name != "" {
		tmuxSession = projectCfg.Name
	}

	// Execute popup
	if err := h.deps.TmuxManager.DisplayPopup(tmuxSession, repoPath, command); err != nil {
		writeError(w, http.StatusInternalServerError, "tmux_error", fmt.Sprintf("failed to display popup: %s", err.Error()))
		return
	}

	resp := ExecuteActionResponse{
		Success: true,
		Message: "Action executed",
	}
	writeJSON(w, http.StatusOK, resp)
}
