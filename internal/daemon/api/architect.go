package api

import (
	"net/http"
	"path/filepath"
	"time"

	"github.com/kareemaly/cortex/internal/core/spawn"
	"github.com/kareemaly/cortex/internal/project/architect"
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

	// Load architect session
	session, err := architect.Load(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "state_error", "failed to load architect state")
		return
	}

	// Detect state
	stateInfo, err := spawn.DetectArchitectState(session, sessionName, h.deps.TmuxManager)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "state_error", "failed to detect architect state")
		return
	}

	resp := ArchitectStateResponse{
		State: string(stateInfo.State),
	}

	if stateInfo.Session != nil {
		resp.Session = &ArchitectSessionResponse{
			ID:          stateInfo.Session.ID,
			TmuxSession: stateInfo.Session.TmuxSession,
			TmuxWindow:  stateInfo.Session.TmuxWindow,
			StartedAt:   stateInfo.Session.StartedAt,
			EndedAt:     stateInfo.Session.EndedAt,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// Spawn handles POST /architect/spawn - spawns an architect session.
func (h *ArchitectHandlers) Spawn(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())

	// Parse mode parameter
	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "normal"
	}
	if mode != "normal" && mode != "resume" && mode != "fresh" {
		writeError(w, http.StatusBadRequest, "invalid_mode", "mode must be normal, resume, or fresh")
		return
	}

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

	// Load architect session
	session, err := architect.Load(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "state_error", "failed to load architect state")
		return
	}

	// Detect state
	stateInfo, err := spawn.DetectArchitectState(session, sessionName, h.deps.TmuxManager)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "state_error", "failed to detect architect state")
		return
	}

	// Apply mode/state matrix
	switch mode {
	case "normal":
		switch stateInfo.State {
		case spawn.StateNormal, spawn.StateEnded:
			// Spawn new
		case spawn.StateActive:
			// Focus window and return existing
			if err := h.deps.TmuxManager.FocusWindow(sessionName, stateInfo.Session.TmuxWindow); err != nil {
				h.deps.Logger.Warn("failed to focus architect window", "error", err)
			}
			resp := ArchitectSpawnResponse{
				State: string(stateInfo.State),
				Session: ArchitectSessionResponse{
					ID:          stateInfo.Session.ID,
					TmuxSession: stateInfo.Session.TmuxSession,
					TmuxWindow:  stateInfo.Session.TmuxWindow,
					StartedAt:   stateInfo.Session.StartedAt,
					EndedAt:     stateInfo.Session.EndedAt,
				},
				TmuxSession: sessionName,
				TmuxWindow:  stateInfo.Session.TmuxWindow,
			}
			writeJSON(w, http.StatusOK, resp)
			return
		case spawn.StateOrphaned:
			writeError(w, http.StatusConflict, "session_orphaned", "architect session is orphaned; use mode=resume or mode=fresh")
			return
		}

	case "resume":
		switch stateInfo.State {
		case spawn.StateOrphaned:
			// Resume - handled below
		case spawn.StateNormal:
			writeError(w, http.StatusBadRequest, "no_session_to_resume", "no architect session to resume")
			return
		case spawn.StateActive:
			writeError(w, http.StatusConflict, "session_active", "architect session is active")
			return
		case spawn.StateEnded:
			writeError(w, http.StatusBadRequest, "session_ended", "architect session has ended; cannot resume")
			return
		}

	case "fresh":
		switch stateInfo.State {
		case spawn.StateOrphaned, spawn.StateEnded:
			// Clear and spawn new - handled below
		case spawn.StateNormal:
			writeError(w, http.StatusBadRequest, "no_session_to_clear", "no architect session to clear")
			return
		case spawn.StateActive:
			writeError(w, http.StatusConflict, "session_active", "architect session is active; cannot clear")
			return
		}
	}

	// Create spawner
	ticketsDir := filepath.Join(projectPath, ".cortex", "tickets")
	spawner := spawn.NewSpawner(spawn.Dependencies{
		TmuxManager: h.deps.TmuxManager,
	})

	var result *spawn.SpawnResult

	if mode == "resume" {
		// Resume orphaned session
		result, err = spawner.Resume(spawn.ResumeRequest{
			AgentType:       spawn.AgentTypeArchitect,
			TmuxSession:     sessionName,
			ProjectPath:     projectPath,
			TicketsDir:      ticketsDir,
			ClaudeSessionID: stateInfo.Session.ID,
			WindowName:      stateInfo.Session.TmuxWindow,
		})
	} else {
		// Fresh mode: clear first
		if mode == "fresh" {
			if err := architect.Clear(projectPath); err != nil {
				h.deps.Logger.Warn("failed to clear architect session", "error", err)
			}
		}

		// Spawn new
		result, err = spawner.Spawn(spawn.SpawnRequest{
			AgentType:   spawn.AgentTypeArchitect,
			Agent:       "claude",
			TmuxSession: sessionName,
			ProjectPath: projectPath,
			TicketsDir:  ticketsDir,
			ProjectName: sessionName,
		})
	}

	if err != nil {
		h.deps.Logger.Error("failed to spawn architect", "error", err)
		writeError(w, http.StatusInternalServerError, "spawn_error", "failed to spawn architect session")
		return
	}

	if !result.Success {
		writeError(w, http.StatusInternalServerError, "spawn_error", result.Message)
		return
	}

	// Save session state
	newSession := &architect.Session{
		ID:          result.SessionID,
		TmuxSession: sessionName,
		TmuxWindow:  result.TmuxWindow,
		StartedAt:   time.Now(),
	}

	// For resume, keep the original session ID if it's empty in result
	if mode == "resume" && newSession.ID == "" && stateInfo.Session != nil {
		newSession.ID = stateInfo.Session.ID
		newSession.StartedAt = stateInfo.Session.StartedAt
	}

	if err := architect.Save(projectPath, newSession); err != nil {
		h.deps.Logger.Error("failed to save architect session", "error", err)
		// Continue anyway - spawn was successful
	}

	resp := ArchitectSpawnResponse{
		State: string(spawn.StateActive),
		Session: ArchitectSessionResponse{
			ID:          newSession.ID,
			TmuxSession: newSession.TmuxSession,
			TmuxWindow:  newSession.TmuxWindow,
			StartedAt:   newSession.StartedAt,
			EndedAt:     newSession.EndedAt,
		},
		TmuxSession: sessionName,
		TmuxWindow:  result.TmuxWindow,
	}

	writeJSON(w, http.StatusCreated, resp)
}
