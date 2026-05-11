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
	architectconfig "github.com/kareemaly/cortex/internal/architect/config"
	"github.com/kareemaly/cortex/internal/core/spawn"
	"github.com/kareemaly/cortex/internal/events"
	"github.com/kareemaly/cortex/internal/storage"
	"github.com/kareemaly/cortex/internal/ticket"
	"github.com/kareemaly/cortex/internal/types"
)

type TicketHandlers struct {
	deps *Dependencies
}

func NewTicketHandlers(deps *Dependencies) *TicketHandlers {
	return &TicketHandlers{deps: deps}
}

func ticketResponse(store *ticket.Store, t *ticket.Ticket, status ticket.Status) (types.TicketResponse, error) {
	hasConclusion := false
	if ok, err := store.HasConclusion(t.ID); err == nil && ok {
		hasConclusion = true
	}
	resp := types.ToTicketResponse(t, status, hasConclusion)

	filePath, err := store.FilePath(t.ID)
	if err != nil {
		return types.TicketResponse{}, err
	}
	resp.FilePath = filePath
	return resp, nil
}

func (h *TicketHandlers) ListAll(w http.ResponseWriter, r *http.Request) {
	projectPath := GetArchitectPath(r.Context())
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

	query := strings.ToLower(r.URL.Query().Get("query"))

	var dueBefore *time.Time
	if dueBeforeStr := r.URL.Query().Get("due_before"); dueBeforeStr != "" {
		parsed, err := time.Parse(time.RFC3339, dueBeforeStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_due_before", "due_before must be in RFC3339 format")
			return
		}
		dueBefore = &parsed
	}

	projectCfg, _ := architectconfig.Load(projectPath)
	tmuxSession := projectCfg.GetTmuxSessionName()

	resp := ListAllTicketsResponse{
		Backlog:  filterSummaryList(all[ticket.StatusBacklog], ticket.StatusBacklog, query, dueBefore, tmuxSession, h.deps.TmuxManager, h.deps.SessionManager, projectPath, h.deps.ReceiverManager, store),
		Progress: filterSummaryList(all[ticket.StatusProgress], ticket.StatusProgress, query, dueBefore, tmuxSession, h.deps.TmuxManager, h.deps.SessionManager, projectPath, h.deps.ReceiverManager, store),
		Done:     filterSummaryList(all[ticket.StatusDone], ticket.StatusDone, query, dueBefore, tmuxSession, h.deps.TmuxManager, h.deps.SessionManager, projectPath, h.deps.ReceiverManager, store),
	}

	sortByCreated := func(a, b TicketSummary) int {
		return b.Created.Compare(a.Created)
	}
	slices.SortFunc(resp.Backlog, sortByCreated)
	slices.SortFunc(resp.Progress, sortByCreated)
	slices.SortFunc(resp.Done, sortByCreated)

	writeJSON(w, http.StatusOK, resp)
}

func (h *TicketHandlers) ListByStatus(w http.ResponseWriter, r *http.Request) {
	status := chi.URLParam(r, "status")
	if !validStatus(status) {
		writeError(w, http.StatusBadRequest, "invalid_status", "invalid status: must be backlog, progress, or done")
		return
	}

	projectPath := GetArchitectPath(r.Context())
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

	query := strings.ToLower(r.URL.Query().Get("query"))

	var dueBefore *time.Time
	if dueBeforeStr := r.URL.Query().Get("due_before"); dueBeforeStr != "" {
		parsed, err := time.Parse(time.RFC3339, dueBeforeStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_due_before", "due_before must be in RFC3339 format")
			return
		}
		dueBefore = &parsed
	}

	projectCfg, _ := architectconfig.Load(projectPath)
	tmuxSession := projectCfg.GetTmuxSessionName()

	resp := ListTicketsResponse{
		Tickets: filterSummaryList(tickets, ticket.Status(status), query, dueBefore, tmuxSession, h.deps.TmuxManager, h.deps.SessionManager, projectPath, h.deps.ReceiverManager, store),
	}

	slices.SortFunc(resp.Tickets, func(a, b TicketSummary) int {
		return b.Created.Compare(a.Created)
	})

	writeJSON(w, http.StatusOK, resp)
}

func (h *TicketHandlers) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	projectPath := GetArchitectPath(r.Context())

	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "missing_title", "title is required")
		return
	}

	var dueDate *time.Time
	if req.DueDate != nil && *req.DueDate != "" {
		parsed, err := time.Parse(time.RFC3339, *req.DueDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_due_date", "due_date must be in RFC3339 format")
			return
		}
		dueDate = &parsed
	}

	projectCfg, _ := architectconfig.Load(projectPath)
	if projectCfg != nil && req.Repo != "" {
		if err := projectCfg.ValidateRepo(req.Repo); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_repo", err.Error())
			return
		}
	}

	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	t, err := store.Create(req.Title, req.Body, dueDate, req.References, req.Repo, req.Path)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	resp, err := ticketResponse(store, t, ticket.StatusBacklog)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (h *TicketHandlers) Get(w http.ResponseWriter, r *http.Request) {
	status := chi.URLParam(r, "status")
	if !validStatus(status) {
		writeError(w, http.StatusBadRequest, "invalid_status", "invalid status: must be backlog, progress, or done")
		return
	}

	projectPath := GetArchitectPath(r.Context())
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

	if string(actualStatus) != status {
		writeError(w, http.StatusNotFound, "not_found", "ticket not found in specified status")
		return
	}

	resp, err := ticketResponse(store, t, actualStatus)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *TicketHandlers) Update(w http.ResponseWriter, r *http.Request) {
	status := chi.URLParam(r, "status")
	if !validStatus(status) {
		writeError(w, http.StatusBadRequest, "invalid_status", "invalid status: must be backlog, progress, or done")
		return
	}

	projectPath := GetArchitectPath(r.Context())
	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	id := chi.URLParam(r, "id")

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

	resp, err := ticketResponse(store, t, actualStatus)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *TicketHandlers) EditBody(w http.ResponseWriter, r *http.Request) {
	projectPath := GetArchitectPath(r.Context())
	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	id := chi.URLParam(r, "id")
	var req EditTicketBodyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	_, status, err := store.Get(id)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	updated, err := store.EditBody(id, req.OldString, req.NewString, req.ReplaceAll)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	resp, err := ticketResponse(store, updated, status)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *TicketHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	status := chi.URLParam(r, "status")
	if !validStatus(status) {
		writeError(w, http.StatusBadRequest, "invalid_status", "invalid status: must be backlog, progress, or done")
		return
	}

	projectPath := GetArchitectPath(r.Context())
	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	id := chi.URLParam(r, "id")

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

func (h *TicketHandlers) Move(w http.ResponseWriter, r *http.Request) {
	status := chi.URLParam(r, "status")
	if !validStatus(status) {
		writeError(w, http.StatusBadRequest, "invalid_status", "invalid status: must be backlog, progress, or done")
		return
	}

	projectPath := GetArchitectPath(r.Context())
	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	id := chi.URLParam(r, "id")

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

	t, newStatus, err := store.Get(id)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	resp, err := ticketResponse(store, t, newStatus)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *TicketHandlers) SetDueDate(w http.ResponseWriter, r *http.Request) {
	projectPath := GetArchitectPath(r.Context())
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

	_, status, err := store.Get(id)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	resp, err := ticketResponse(store, t, status)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *TicketHandlers) ClearDueDate(w http.ResponseWriter, r *http.Request) {
	projectPath := GetArchitectPath(r.Context())
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

	_, status, err := store.Get(id)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	resp, err := ticketResponse(store, t, status)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *TicketHandlers) Spawn(w http.ResponseWriter, r *http.Request) {
	status := chi.URLParam(r, "status")
	if !validStatus(status) {
		writeError(w, http.StatusBadRequest, "invalid_status", "invalid status: must be backlog, progress, or done")
		return
	}

	id := chi.URLParam(r, "id")
	mode := r.URL.Query().Get("mode")
	variantName := r.URL.Query().Get("variant")
	projectPath := GetArchitectPath(r.Context())

	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	_, actualStatus, err := store.Get(id)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}
	if string(actualStatus) != status {
		writeError(w, http.StatusNotFound, "not_found", "ticket not found in specified status")
		return
	}

	if h.deps.TmuxManager == nil {
		writeError(w, http.StatusServiceUnavailable, "tmux_unavailable", "tmux is not installed")
		return
	}

	projectCfg, _ := mergeProjectConfig(projectPath)

	if variantName == "" {
		names := projectCfg.VariantNames()
		var msg string
		if len(names) > 0 {
			msg = fmt.Sprintf("--variant is required, choose one of: %s", strings.Join(names, ", "))
		} else {
			msg = "--variant is required — run 'cortex init' to populate defaults in ~/.cortex/settings.yaml"
		}
		writeError(w, http.StatusBadRequest, "variant_required", msg)
		return
	}
	av, avErr := projectCfg.ResolveVariant(variantName)
	if avErr != nil {
		writeError(w, http.StatusBadRequest, "invalid_variant", avErr.Error())
		return
	}
	resolvedAgent := string(av.Agent)
	if resolvedAgent == "" {
		resolvedAgent = "claude"
	}

	sessionStore := h.deps.SessionManager.GetStore(projectPath)
	result, err := spawn.Orchestrate(r.Context(), spawn.OrchestrateRequest{
		TicketID:      id,
		Mode:          mode,
		Agent:         resolvedAgent,
		AgentArgs:     av.Args,
		EnvVars:       av.Env,
		Companion:     projectCfg.Companion,
		ArchitectPath: projectPath,
	}, spawn.OrchestrateDeps{
		Store:          store,
		SessionStore:   sessionStore,
		TmuxManager:    h.deps.TmuxManager,
		SupervisorCtx:  h.deps.SupervisorCtx,
		Logger:         h.deps.Logger,
		CortexdPath:    h.deps.CortexdPath,
		DefaultsDir:    h.deps.DefaultsDir,
		HubEventSource: hubEventSource(h.deps.ReceiverManager),
		DaemonEndpoint: h.deps.DaemonEndpoint,
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

	if result.Outcome == spawn.OutcomeAlreadyActive {
		ticketResp, err := ticketResponse(store, result.Ticket, result.TicketStatus)
		if err != nil {
			handleTicketError(w, err, h.deps.Logger)
			return
		}
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
			Ticket:  ticketResp,
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}

	sess, _ := sessionStore.GetByTicketID(id)
	ticketResp, err := ticketResponse(store, result.Ticket, result.TicketStatus)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}
	resp := SpawnResponse{
		Ticket: ticketResp,
	}
	if sess != nil {
		resp.Session = types.ToSessionResponse(sess)
	}
	h.deps.Bus.Emit(events.Event{
		Type:          events.SessionStarted,
		ArchitectPath: projectPath,
		TicketID:      id,
	})
	writeJSON(w, http.StatusCreated, resp)
}

func (h *TicketHandlers) GetDiffs(w http.ResponseWriter, r *http.Request) {
	projectPath := GetArchitectPath(r.Context())
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

	hasConclusion, _ := store.HasConclusion(id)
	if !hasConclusion {
		writeError(w, http.StatusNotFound, "no_conclusion", "ticket has no conclusion")
		return
	}

	concMeta, _, err := store.ReadConclusion(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "no_conclusion", "failed to read ticket conclusion")
		return
	}

	repoDir, err := resolveGitRepoDir(t.Repo)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_repo", err.Error())
		return
	}

	resp := DiffsResponse{
		TicketID: t.ID,
		Repo:     repoDir,
		Commits:  []CommitDiffResponse{},
	}
	if len(concMeta.Commits) == 0 {
		writeJSON(w, http.StatusOK, resp)
		return
	}

	if invalid := validateCommitSHAs(repoDir, concMeta.Commits); len(invalid) > 0 {
		writeError(w, http.StatusNotFound, "commit_not_found",
			fmt.Sprintf("commit %s does not exist in %s", invalid[0], repoDir))
		return
	}

	resp.Commits = make([]CommitDiffResponse, 0, len(concMeta.Commits))
	for _, sha := range concMeta.Commits {
		diff, err := buildCommitDiff(repoDir, sha)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "git_error", err.Error())
			return
		}
		resp.Commits = append(resp.Commits, *diff)
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *TicketHandlers) GetByID(w http.ResponseWriter, r *http.Request) {
	projectPath := GetArchitectPath(r.Context())
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

	resp, err := ticketResponse(store, t, status)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *TicketHandlers) Focus(w http.ResponseWriter, r *http.Request) {
	projectPath := GetArchitectPath(r.Context())
	id := chi.URLParam(r, "id")

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

	projectCfg, _ := architectconfig.Load(projectPath)
	tmuxSession := projectCfg.GetTmuxSessionName()

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

func (h *TicketHandlers) Conclude(w http.ResponseWriter, r *http.Request) {
	projectPath := GetArchitectPath(r.Context())
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

	repo := t.Repo

	if !req.Rejected {
		if len(req.Commits) == 0 {
			writeError(w, http.StatusBadRequest, "validation_error",
				"Cannot conclude: commits array is required with at least one SHA. If this session produced no work, pass rejected: true with a rejection_reason.")
			return
		}
		if invalid := validateCommitSHAs(repo, req.Commits); len(invalid) > 0 {
			writeError(w, http.StatusBadRequest, "validation_error",
				"Cannot conclude: commit "+invalid[0]+" does not exist in "+repo+".")
			return
		}
	} else {
		if req.RejectionReason == "" {
			writeError(w, http.StatusBadRequest, "validation_error",
				"Cannot conclude: rejection_reason is required when rejected: true.")
			return
		}
		if invalid := validateCommitSHAs(repo, req.Commits); len(invalid) > 0 {
			writeError(w, http.StatusBadRequest, "validation_error",
				"Cannot conclude: commit "+invalid[0]+" does not exist in "+repo+".")
			return
		}
	}

	var tmuxWindow string
	var agent string
	if h.deps.SessionManager != nil {
		sessStore := h.deps.SessionManager.GetStore(projectPath)
		if sess, sessErr := sessStore.GetByTicketID(id); sessErr == nil && sess != nil {
			tmuxWindow = sess.TmuxWindow
			agent = sess.Agent
		}
	}

	if h.deps.SessionManager != nil {
		sessStore := h.deps.SessionManager.GetStore(projectPath)
		if endErr := sessStore.EndByTicketID(id); endErr != nil && !storage.IsNotFound(endErr) {
			h.deps.Logger.Warn("failed to end session", "error", endErr)
		}
	}

	h.deps.Bus.Emit(events.Event{
		Type:          events.SessionEnded,
		ArchitectPath: projectPath,
		TicketID:      id,
	})

	var startedAt time.Time
	if req.StartedAt != "" {
		if parsed, parseErr := time.Parse(time.RFC3339, req.StartedAt); parseErr == nil {
			startedAt = parsed
		}
	}
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}

	concludedAt := time.Now().UTC()

	conclusionMeta := &ticket.TicketConclusionMeta{
		StartedAt:       startedAt,
		ConcludedAt:     concludedAt,
		Agent:           agent,
		Profile:         "",
		Rejected:        req.Rejected,
		RejectionReason: req.RejectionReason,
		Commits:         req.Commits,
	}

	if writeErr := store.WriteConclusion(id, conclusionMeta, req.Content); writeErr != nil {
		h.deps.Logger.Warn("failed to write conclusion", "error", writeErr)
	}

	if err := store.Move(id, ticket.StatusDone); err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	if tmuxWindow != "" && h.deps.TmuxManager != nil {
		projectCfg, _ := architectconfig.Load(projectPath)
		tmuxSession := projectCfg.GetTmuxSessionName()
		if killErr := h.deps.TmuxManager.KillWindow(tmuxSession, tmuxWindow); killErr != nil {
			h.deps.Logger.Warn("failed to kill tmux window", "window", tmuxWindow, "error", killErr)
		}
	}

	resp := ConcludeSessionResponse{
		Success:  true,
		TicketID: id,
		Message:  "Session concluded and ticket moved to done",
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *TicketHandlers) Show(w http.ResponseWriter, r *http.Request) {
	projectPath := GetArchitectPath(r.Context())
	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	ticketID := chi.URLParam(r, "id")
	if _, _, err := store.Get(ticketID); err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	if h.deps.TmuxManager == nil {
		writeError(w, http.StatusServiceUnavailable, "tmux_unavailable", "tmux is not installed")
		return
	}

	if err := openCortexPopup(projectPath, h.deps.TmuxManager, "ticket", "show", ticketID); err != nil {
		writeError(w, http.StatusInternalServerError, "tmux_error", fmt.Sprintf("failed to display popup: %s", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, ExecuteActionResponse{
		Success: true,
		Message: "Ticket viewer opened",
	})
}

func (h *TicketHandlers) Edit(w http.ResponseWriter, r *http.Request) {
	projectPath := GetArchitectPath(r.Context())
	store, err := h.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	ticketID := chi.URLParam(r, "id")

	filePath, err := store.FilePath(ticketID)
	if err != nil {
		handleTicketError(w, err, h.deps.Logger)
		return
	}

	if h.deps.TmuxManager == nil {
		writeError(w, http.StatusServiceUnavailable, "tmux_unavailable", "tmux is not installed")
		return
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	command := fmt.Sprintf("%s %q", editor, filePath)

	projectCfg, _ := architectconfig.Load(projectPath)
	tmuxSession := projectCfg.GetTmuxSessionName()

	if err := h.deps.TmuxManager.DisplayPopup(tmuxSession, "", command); err != nil {
		writeError(w, http.StatusInternalServerError, "tmux_error", fmt.Sprintf("failed to display popup: %s", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, ExecuteActionResponse{
		Success: true,
		Message: "Editor opened",
	})
}
