package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	architectconfig "github.com/kareemaly/cortex/internal/architect/config"
	"github.com/kareemaly/cortex/internal/core/spawn"
	"github.com/kareemaly/cortex/internal/events"
	"github.com/kareemaly/cortex/internal/types"
)

// SpawnCollabRequest is the request body for spawning a collab session.
type SpawnCollabRequest struct {
	Repo   string `json:"repo"`
	Prompt string `json:"prompt"`
	Mode   string `json:"mode,omitempty"`
}

// CollabHandlers provides HTTP handlers for collab session operations.
type CollabHandlers struct {
	deps *Dependencies
}

// NewCollabHandlers creates a new CollabHandlers with the given dependencies.
func NewCollabHandlers(deps *Dependencies) *CollabHandlers {
	return &CollabHandlers{deps: deps}
}

// Spawn handles POST /collab/spawn - spawns a collab session.
func (h *CollabHandlers) Spawn(w http.ResponseWriter, r *http.Request) {
	projectPath := GetArchitectPath(r.Context())

	var req SpawnCollabRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	if req.Repo == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "repo cannot be empty")
		return
	}
	if req.Prompt == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "prompt cannot be empty")
		return
	}

	// Load project config
	projectCfg, err := architectconfig.Load(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "config_error", "failed to load project config")
		return
	}

	// Validate repo
	if err := projectCfg.ValidateRepo(req.Repo); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	// Check tmux is available
	if h.deps.TmuxManager == nil {
		writeError(w, http.StatusServiceUnavailable, "tmux_unavailable", "tmux is not installed")
		return
	}

	sessionName := projectCfg.GetTmuxSessionName()

	collabID := uuid.New().String()

	collabAgent := string(projectCfg.Collab.Agent)
	if collabAgent == "" {
		collabAgent = "claude"
	}

	ticketsDir := projectCfg.TicketsPath(projectPath)

	var sessStore spawn.SessionStoreInterface
	if h.deps.SessionManager != nil {
		sessStore = h.deps.SessionManager.GetStore(projectPath)
	}

	spawner := spawn.NewSpawner(spawn.Dependencies{
		TmuxManager:  h.deps.TmuxManager,
		SessionStore: sessStore,
		CortexdPath:  h.deps.CortexdPath,
		Logger:       h.deps.Logger,
		DefaultsDir:  h.deps.DefaultsDir,
	})

	result, err := spawner.SpawnCollab(r.Context(), spawn.CollabSpawnRequest{
		CollabID:      collabID,
		Repo:          req.Repo,
		Prompt:        req.Prompt,
		ArchitectPath: projectPath,
		TmuxSession:   sessionName,
		Agent:         collabAgent,
		Companion:     projectCfg.Collab.Companion,
		AgentArgs:     projectCfg.Collab.Args,
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
		CollabID:    result.CollabID,
		TmuxWindow:  result.TmuxWindow,
		TmuxSession: result.TmuxSession,
		State:       "active",
	})
}

// Conclude handles POST /collab/{id}/conclude - concludes a collab session.
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

	// Get session info before ending
	var tmuxWindow string
	var prompt string
	var repo string
	if h.deps.SessionManager != nil {
		sessStore := h.deps.SessionManager.GetStore(projectPath)
		if sess, err := sessStore.GetByCollabID(collabID); err == nil && sess != nil {
			tmuxWindow = sess.TmuxWindow
			prompt = sess.Prompt
			repo = sess.CollabID // repo is not stored in session; use Repo from req if available
			_ = repo
		}
		// End the ephemeral session
		if endErr := sessStore.EndCollab(collabID); endErr != nil {
			h.deps.Logger.Warn("failed to end collab session", "error", endErr)
		}
	}

	// Use repo from request if provided
	if req.Repo != "" {
		repo = req.Repo
	}

	h.deps.Bus.Emit(events.Event{
		Type:          events.SessionEnded,
		ArchitectPath: projectPath,
	})

	// Parse startedAt
	var startedAt time.Time
	if req.StartedAt != "" {
		if parsed, parseErr := time.Parse(time.RFC3339, req.StartedAt); parseErr == nil {
			startedAt = parsed
		}
	}

	// Create conclusion record and kill tmux window
	CreateConclusionAndKillWindow(ConcludeParams{
		ProjectPath:   projectPath,
		EntityType:    "collab",
		EntityID:      collabID,
		TmuxWindow:    tmuxWindow,
		Content:       req.Content,
		StartedAt:     startedAt,
		Repo:          repo,
		Prompt:        prompt,
		Logger:        h.deps.Logger,
		TmuxManager:   h.deps.TmuxManager,
		ConclusionMgr: h.deps.ConclusionStoreManager,
	})

	writeJSON(w, http.StatusOK, ConcludeSessionResponse{
		Success:  true,
		TicketID: collabID,
		Message:  "Collab session concluded",
	})
}

// Focus handles POST /collab/{id}/focus - focuses the tmux window of a collab session.
func (h *CollabHandlers) Focus(w http.ResponseWriter, r *http.Request) {
	projectPath := GetArchitectPath(r.Context())
	sessionID := chi.URLParam(r, "id")

	// Look up session from session manager by short ID (the key in the session store)
	if h.deps.SessionManager == nil {
		writeError(w, http.StatusNotFound, "no_active_session", "no session manager available")
		return
	}

	sessStore := h.deps.SessionManager.GetStore(projectPath)
	sess, err := sessStore.Get(sessionID)
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
