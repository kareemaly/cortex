package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	architectconfig "github.com/kareemaly/cortex/internal/architect/config"
	"github.com/kareemaly/cortex/internal/collab"
	"github.com/kareemaly/cortex/internal/core/spawn"
	"github.com/kareemaly/cortex/internal/events"
	"github.com/kareemaly/cortex/internal/types"
)

type CollabHandlers struct {
	deps *Dependencies
}

func NewCollabHandlers(deps *Dependencies) *CollabHandlers {
	return &CollabHandlers{deps: deps}
}

func (h *CollabHandlers) Spawn(w http.ResponseWriter, r *http.Request) {
	projectPath := GetArchitectPath(r.Context())

	var req SpawnCollabRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	if req.Path == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "path cannot be empty")
		return
	}
	if req.Prompt == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "prompt cannot be empty")
		return
	}
	if req.Slug == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "slug is required")
		return
	}

	expandedPath := req.Path
	if strings.HasPrefix(expandedPath, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			expandedPath = filepath.Join(home, expandedPath[2:])
		}
	}
	if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
		writeError(w, http.StatusBadRequest, "validation_error", fmt.Sprintf("path %q does not exist", req.Path))
		return
	}

	projectCfg, err := mergeProjectConfig(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "config_error", "failed to load project config")
		return
	}

	if h.deps.TmuxManager == nil {
		writeError(w, http.StatusServiceUnavailable, "tmux_unavailable", "tmux is not installed")
		return
	}

	sessionName := projectCfg.GetTmuxSessionName()

	variantName := req.Variant
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
	collabAgent := string(av.Agent)
	if collabAgent == "" {
		collabAgent = "claude"
	}

	if err := collab.EnsureDir(projectPath); err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", "failed to create collabs directory")
		return
	}

	now := time.Now().UTC()
	collabID, idErr := collab.NewID(collab.Dir(projectPath), now, req.Slug)
	if idErr != nil {
		writeError(w, http.StatusInternalServerError, "store_error", "failed to generate collab ID")
		return
	}

	promptMeta := collab.PromptMeta{
		Created: now,
		Agent:   collabAgent,
		Profile: variantName,
	}
	if err := collab.WritePrompt(projectPath, collabID, promptMeta, req.Prompt); err != nil {
		h.deps.Logger.Warn("failed to persist collab prompt", "error", err)
	}

	ticketsDir := projectCfg.TicketsPath(projectPath)

	var sessStore spawn.SessionStoreInterface
	if h.deps.SessionManager != nil {
		sessStore = h.deps.SessionManager.GetStore(projectPath)
	}

	spawner := spawn.NewSpawner(spawn.Dependencies{
		TmuxManager:    h.deps.TmuxManager,
		SessionStore:   sessStore,
		SupervisorCtx:  h.deps.SupervisorCtx,
		CortexdPath:    h.deps.CortexdPath,
		Logger:         h.deps.Logger,
		DefaultsDir:    h.deps.DefaultsDir,
		HubEventSource: hubEventSource(h.deps.ReceiverManager),
		DaemonEndpoint: h.deps.DaemonEndpoint,
	})

	result, err := spawner.SpawnCollab(r.Context(), spawn.CollabSpawnRequest{
		CollabID:      collabID,
		Repo:          expandedPath,
		Prompt:        req.Prompt,
		ArchitectPath: projectPath,
		TmuxSession:   sessionName,
		Agent:         collabAgent,
		Companion:     projectCfg.Companion,
		AgentArgs:     av.Args,
		EnvVars:       av.Env,
		TicketsDir:    ticketsDir,
	})
	if err != nil {
		h.deps.Logger.Error("failed to spawn collab session", "error", err)
		writeError(w, http.StatusInternalServerError, "spawn_error", "failed to spawn collab session: "+err.Error())
		return
	}

	h.deps.Bus.Emit(events.Event{
		Type:          events.SessionStarted,
		ArchitectPath: projectPath,
	})

	writeJSON(w, http.StatusCreated, types.SpawnCollabResponse{
		CollabID:    collabID,
		TmuxWindow:  result.TmuxWindow,
		TmuxSession: result.TmuxSession,
		State:       "active",
	})
}

func (h *CollabHandlers) Conclude(w http.ResponseWriter, r *http.Request) {
	projectPath := GetArchitectPath(r.Context())
	collabID := chi.URLParam(r, "id")

	var req ConcludeSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "content cannot be empty")
		return
	}

	var tmuxWindow string
	var agent string
	if h.deps.SessionManager != nil {
		sessStore := h.deps.SessionManager.GetStore(projectPath)
		if sess, err := sessStore.GetByCollabID(collabID); err == nil && sess != nil {
			tmuxWindow = sess.TmuxWindow
			agent = sess.Agent
		}
		if endErr := sessStore.EndCollab(collabID); endErr != nil {
			h.deps.Logger.Warn("failed to end collab session", "error", endErr)
		}
	}

	h.deps.Bus.Emit(events.Event{
		Type:          events.SessionEnded,
		ArchitectPath: projectPath,
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
	concMeta := collab.ConclusionMeta{
		StartedAt:   startedAt,
		ConcludedAt: concludedAt,
		Agent:       agent,
		Profile:     "",
	}

	if err := collab.WriteConclusion(projectPath, collabID, concMeta, req.Content); err != nil {
		h.deps.Logger.Warn("failed to write collab conclusion", "error", err)
	}

	if tmuxWindow != "" && h.deps.TmuxManager != nil {
		projectCfg, _ := architectconfig.Load(projectPath)
		tmuxSession := projectCfg.GetTmuxSessionName()
		if killErr := h.deps.TmuxManager.KillWindow(tmuxSession, tmuxWindow); killErr != nil {
			h.deps.Logger.Warn("failed to kill tmux window", "window", tmuxWindow, "error", killErr)
		}
	}

	writeJSON(w, http.StatusOK, ConcludeSessionResponse{
		Success:  true,
		TicketID: collabID,
		Message:  "Collab session concluded",
	})
}

func (h *CollabHandlers) Focus(w http.ResponseWriter, r *http.Request) {
	projectPath := GetArchitectPath(r.Context())
	sessionID := chi.URLParam(r, "id")

	if h.deps.SessionManager == nil {
		writeError(w, http.StatusServiceUnavailable, "sessions_unavailable",
			"session manager is not configured")
		return
	}

	sessStore := h.deps.SessionManager.GetStore(projectPath)
	sess, err := sessStore.GetBySessionID(sessionID)
	if err != nil || sess == nil || sess.TmuxWindow == "" {
		writeError(w, http.StatusNotFound, "no_active_session", "collab has no active session with a tmux window")
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
	_ = json.NewEncoder(w).Encode(resp)
}
