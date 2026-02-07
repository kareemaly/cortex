package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"time"

	"github.com/kareemaly/cortex/internal/core/spawn"
	projectconfig "github.com/kareemaly/cortex/internal/project/config"
	"github.com/kareemaly/cortex/internal/session"
	"github.com/kareemaly/cortex/internal/tmux"
)

// ArchitectHandlers provides HTTP handlers for architect session operations.
type ArchitectHandlers struct {
	deps *Dependencies
}

// NewArchitectHandlers creates a new ArchitectHandlers with the given dependencies.
func NewArchitectHandlers(deps *Dependencies) *ArchitectHandlers {
	return &ArchitectHandlers{deps: deps}
}

// getSessionAndConfig is a helper that loads project config and retrieves the architect session.
func (h *ArchitectHandlers) getSessionAndConfig(projectPath string) (sessionName string, sess *session.Session, projectCfg *projectconfig.Config, err error) {
	projectCfg, err = projectconfig.Load(projectPath)
	if err != nil {
		return "", nil, nil, err
	}

	sessionName = projectCfg.Name
	if sessionName == "" {
		sessionName = "cortex"
	}

	if h.deps.SessionManager != nil {
		sessStore := h.deps.SessionManager.GetStore(projectPath)
		sess, _ = sessStore.GetArchitect()
	}

	return sessionName, sess, projectCfg, nil
}

// GetState handles GET /architect - returns architect session state.
func (h *ArchitectHandlers) GetState(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())

	sessionName, sess, _, err := h.getSessionAndConfig(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "config_error", "failed to load project config")
		return
	}

	// Detect state using session store + tmux
	stateInfo, err := spawn.DetectArchitectState(sess, sessionName, h.deps.TmuxManager)
	if err != nil {
		h.deps.Logger.Warn("failed to detect architect state", "error", err)
		// Fall back to basic state
		writeJSON(w, http.StatusOK, ArchitectStateResponse{State: "normal"})
		return
	}

	resp := ArchitectStateResponse{
		State: string(stateInfo.State),
	}

	if stateInfo.State != spawn.StateNormal {
		archResp := &ArchitectSessionResponse{
			TmuxSession: sessionName,
			TmuxWindow:  "architect",
		}
		if sess != nil {
			archResp.StartedAt = sess.StartedAt
			status := string(sess.Status)
			archResp.Status = &status
			archResp.Tool = sess.Tool
			archResp.IsOrphaned = stateInfo.State == spawn.StateOrphaned
		}
		resp.Session = archResp
	}

	writeJSON(w, http.StatusOK, resp)
}

// Spawn handles POST /architect/spawn - spawns an architect session.
func (h *ArchitectHandlers) Spawn(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())
	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "normal"
	}

	sessionName, sess, projectCfg, err := h.getSessionAndConfig(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "config_error", "failed to load project config")
		return
	}

	// Check tmux is available
	if h.deps.TmuxManager == nil {
		writeError(w, http.StatusServiceUnavailable, "tmux_unavailable", "tmux is not installed")
		return
	}

	// Detect state
	stateInfo, err := spawn.DetectArchitectState(sess, sessionName, h.deps.TmuxManager)
	if err != nil {
		h.deps.Logger.Error("failed to detect architect state", "error", err)
		writeError(w, http.StatusInternalServerError, "state_error", "failed to detect architect session state")
		return
	}

	switch stateInfo.State {
	case spawn.StateActive:
		// Focus the existing window and return 200
		if err := h.deps.TmuxManager.FocusWindow(sessionName, "architect"); err != nil {
			h.deps.Logger.Warn("failed to focus architect window", "error", err)
		}
		if err := h.deps.TmuxManager.SwitchClient(sessionName); err != nil {
			h.deps.Logger.Warn("failed to switch tmux client", "session", sessionName, "error", err)
		}
		archResp := ArchitectSessionResponse{
			TmuxSession: sessionName,
			TmuxWindow:  "architect",
		}
		if sess != nil {
			archResp.StartedAt = sess.StartedAt
			status := string(sess.Status)
			archResp.Status = &status
			archResp.Tool = sess.Tool
		}
		writeJSON(w, http.StatusOK, ArchitectSpawnResponse{
			State:       "active",
			Session:     archResp,
			TmuxSession: sessionName,
			TmuxWindow:  "architect",
		})
		return

	case spawn.StateOrphaned:
		switch mode {
		case "normal":
			// Return 409 — client should choose fresh or resume
			writeError(w, http.StatusConflict, "session_orphaned",
				"architect session was orphaned (tmux window closed). Use mode=fresh to start over or mode=resume to continue")
			return
		case "fresh":
			// End old session, then spawn new
			if h.deps.SessionManager != nil {
				sessStore := h.deps.SessionManager.GetStore(projectPath)
				if endErr := sessStore.EndArchitect(); endErr != nil {
					h.deps.Logger.Warn("failed to end orphaned architect session", "error", endErr)
				}
			}
			// Fall through to spawn below
		case "resume":
			// End old session record, spawn with --resume
			if h.deps.SessionManager != nil {
				sessStore := h.deps.SessionManager.GetStore(projectPath)
				if endErr := sessStore.EndArchitect(); endErr != nil {
					h.deps.Logger.Warn("failed to end orphaned architect session for resume", "error", endErr)
				}
			}
			h.spawnArchitectSession(w, r, projectPath, sessionName, projectCfg, true)
			return
		default:
			writeError(w, http.StatusBadRequest, "invalid_mode", "mode must be 'normal', 'fresh', or 'resume'")
			return
		}

	case spawn.StateNormal:
		if mode != "normal" && mode != "" {
			writeError(w, http.StatusBadRequest, "invalid_mode",
				"cannot use mode '"+mode+"' when no existing architect session exists")
			return
		}
		// Fall through to spawn below
	}

	// Spawn a new architect session
	h.spawnArchitectSession(w, r, projectPath, sessionName, projectCfg, false)
}

// spawnArchitectSession spawns an architect session (new or resumed).
func (h *ArchitectHandlers) spawnArchitectSession(w http.ResponseWriter, r *http.Request, projectPath, sessionName string, projectCfg *projectconfig.Config, resume bool) {
	ticketsDir := filepath.Join(projectPath, ".cortex", "tickets")

	var sessStore spawn.SessionStoreInterface
	if h.deps.SessionManager != nil {
		sessStore = h.deps.SessionManager.GetStore(projectPath)
	}

	spawner := spawn.NewSpawner(spawn.Dependencies{
		TmuxManager:  h.deps.TmuxManager,
		SessionStore: sessStore,
		CortexdPath:  h.deps.CortexdPath,
		Logger:       h.deps.Logger,
	})

	architectAgent := string(projectCfg.Architect.Agent)
	if architectAgent == "" {
		architectAgent = "claude"
	}

	var result *spawn.SpawnResult
	var err error

	if resume {
		result, err = spawner.Resume(r.Context(), spawn.ResumeRequest{
			AgentType:   spawn.AgentTypeArchitect,
			Agent:       architectAgent,
			TmuxSession: sessionName,
			ProjectPath: projectPath,
			TicketsDir:  ticketsDir,
			WindowName:  "architect",
			AgentArgs:   projectCfg.Architect.Args,
		})
	} else {
		result, err = spawner.Spawn(r.Context(), spawn.SpawnRequest{
			AgentType:      spawn.AgentTypeArchitect,
			Agent:          architectAgent,
			TmuxSession:    sessionName,
			ProjectPath:    projectPath,
			TicketsDir:     ticketsDir,
			ProjectName:    sessionName,
			AgentArgs:      projectCfg.Architect.Args,
			BaseConfigPath: projectCfg.ResolvedExtendPath(),
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

	writeJSON(w, http.StatusCreated, ArchitectSpawnResponse{
		State: "active",
		Session: ArchitectSessionResponse{
			TmuxSession: sessionName,
			TmuxWindow:  result.TmuxWindow,
		},
		TmuxSession: sessionName,
		TmuxWindow:  result.TmuxWindow,
	})
}

// Conclude handles POST /architect/conclude - concludes the architect session.
func (h *ArchitectHandlers) Conclude(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())

	var req ConcludeSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "content cannot be empty")
		return
	}

	// Get architect session info before ending
	var tmuxWindow string
	if h.deps.SessionManager != nil {
		sessStore := h.deps.SessionManager.GetStore(projectPath)
		if sess, err := sessStore.GetArchitect(); err == nil && sess != nil {
			tmuxWindow = sess.TmuxWindow
		}

		// End the session
		if endErr := sessStore.EndArchitect(); endErr != nil {
			h.deps.Logger.Warn("failed to end architect session", "error", endErr)
		}
	}

	// Persist session summary as a doc (best-effort)
	if h.deps.DocsStoreManager != nil {
		docStore, err := h.deps.DocsStoreManager.GetStore(projectPath)
		if err != nil {
			h.deps.Logger.Warn("failed to get docs store for session summary", "error", err)
		} else {
			title := "Architect Session — " + time.Now().UTC().Format("2006-01-02T15:04Z")
			tags := []string{"architect", "session-summary"}
			if _, err := docStore.Create(title, "sessions", req.Content, tags, nil); err != nil {
				h.deps.Logger.Warn("failed to persist architect session summary", "error", err)
			}
		}
	}

	// Kill tmux window if associated (best-effort)
	if tmuxWindow != "" && h.deps.TmuxManager != nil {
		projectCfg, cfgErr := projectconfig.Load(projectPath)
		tmuxSession := "cortex"
		if cfgErr == nil && projectCfg.Name != "" {
			tmuxSession = projectCfg.Name
		}

		if err := h.deps.TmuxManager.KillWindow(tmuxSession, tmuxWindow); err != nil {
			if !tmux.IsWindowNotFound(err) && !tmux.IsSessionNotFound(err) {
				h.deps.Logger.Warn("failed to kill architect tmux window", "error", err)
			}
		}
	}

	writeJSON(w, http.StatusOK, ConcludeSessionResponse{
		Success:  true,
		TicketID: session.ArchitectSessionKey,
		Message:  "Architect session concluded",
	})
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
