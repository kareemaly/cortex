package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"github.com/kareemaly/cortex/internal/core/spawn"
	projectconfig "github.com/kareemaly/cortex/internal/project/config"
)

// ArchitectHandlers provides HTTP handlers for architect session operations.
type ArchitectHandlers struct {
	deps *Dependencies
}

// NewArchitectHandlers creates a new ArchitectHandlers with the given dependencies.
func NewArchitectHandlers(deps *Dependencies) *ArchitectHandlers {
	return &ArchitectHandlers{deps: deps}
}

// GetState handles GET /architect - returns architect session state.
func (h *ArchitectHandlers) GetState(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())

	// Load project config to get session name
	projectCfg, err := projectconfig.Load(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "config_error", "failed to load project config")
		return
	}

	sessionName := projectCfg.Name
	if sessionName == "" {
		sessionName = "cortex"
	}

	// Check if architect window exists
	windowExists := false
	if h.deps.TmuxManager != nil {
		exists, err := h.deps.TmuxManager.WindowExists(sessionName, "architect")
		if err == nil {
			windowExists = exists
		}
	}

	state := "normal"
	if windowExists {
		state = "active"
	}

	resp := ArchitectStateResponse{
		State: state,
	}

	if windowExists {
		resp.Session = &ArchitectSessionResponse{
			TmuxSession: sessionName,
			TmuxWindow:  "architect",
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// Spawn handles POST /architect/spawn - spawns an architect session.
func (h *ArchitectHandlers) Spawn(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())

	// Load project config
	projectCfg, err := projectconfig.Load(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "config_error", "failed to load project config")
		return
	}

	sessionName := projectCfg.Name
	if sessionName == "" {
		sessionName = "cortex"
	}

	// Check tmux is available
	if h.deps.TmuxManager == nil {
		writeError(w, http.StatusServiceUnavailable, "tmux_unavailable", "tmux is not installed")
		return
	}

	// Check if architect window already exists
	windowExists, err := h.deps.TmuxManager.WindowExists(sessionName, "architect")
	if err != nil {
		h.deps.Logger.Warn("failed to check architect window existence", "error", err)
	}

	// If window exists, focus it and return
	if windowExists {
		if err := h.deps.TmuxManager.FocusWindow(sessionName, "architect"); err != nil {
			h.deps.Logger.Warn("failed to focus architect window", "error", err)
		}
		if err := h.deps.TmuxManager.SwitchClient(sessionName); err != nil {
			h.deps.Logger.Warn("failed to switch tmux client", "session", sessionName, "error", err)
		}
		resp := ArchitectSpawnResponse{
			State: "active",
			Session: ArchitectSessionResponse{
				TmuxSession: sessionName,
				TmuxWindow:  "architect",
			},
			TmuxSession: sessionName,
			TmuxWindow:  "architect",
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}

	// Spawn fresh architect session
	ticketsDir := filepath.Join(projectPath, ".cortex", "tickets")
	spawner := spawn.NewSpawner(spawn.Dependencies{
		TmuxManager: h.deps.TmuxManager,
	})

	architectAgent := string(projectCfg.Architect.Agent)
	if architectAgent == "" {
		architectAgent = "claude"
	}

	result, err := spawner.Spawn(r.Context(), spawn.SpawnRequest{
		AgentType:      spawn.AgentTypeArchitect,
		Agent:          architectAgent,
		TmuxSession:    sessionName,
		ProjectPath:    projectPath,
		TicketsDir:     ticketsDir,
		ProjectName:    sessionName,
		AgentArgs:      projectCfg.Architect.Args,
		BaseConfigPath: projectCfg.ResolvedExtendPath(),
	})

	if err != nil {
		h.deps.Logger.Error("failed to spawn architect", "error", err)
		writeError(w, http.StatusInternalServerError, "spawn_error", "failed to spawn architect session")
		return
	}

	if !result.Success {
		writeError(w, http.StatusInternalServerError, "spawn_error", result.Message)
		return
	}

	resp := ArchitectSpawnResponse{
		State: "active",
		Session: ArchitectSessionResponse{
			TmuxSession: sessionName,
			TmuxWindow:  result.TmuxWindow,
		},
		TmuxSession: sessionName,
		TmuxWindow:  result.TmuxWindow,
	}

	writeJSON(w, http.StatusCreated, resp)
}

// Focus handles POST /architect/focus - focuses the architect tmux window.
func (h *ArchitectHandlers) Focus(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())

	projectCfg, err := projectconfig.Load(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "config_error", "failed to load project config")
		return
	}

	sessionName := projectCfg.Name
	if sessionName == "" {
		sessionName = "cortex"
	}

	if h.deps.TmuxManager == nil {
		writeError(w, http.StatusServiceUnavailable, "tmux_unavailable", "tmux is not installed")
		return
	}

	if err := h.deps.TmuxManager.FocusWindowByIndex(sessionName, 0); err != nil {
		writeError(w, http.StatusInternalServerError, "focus_error", err.Error())
		return
	}

	if err := h.deps.TmuxManager.SwitchClient(sessionName); err != nil {
		h.deps.Logger.Warn("failed to switch tmux client", "session", sessionName, "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(FocusResponse{Success: true, Window: "architect"})
}
