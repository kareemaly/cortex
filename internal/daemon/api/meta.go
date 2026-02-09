package api

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"path/filepath"

	"github.com/kareemaly/cortex/internal/core/spawn"
	"github.com/kareemaly/cortex/internal/session"
	"github.com/kareemaly/cortex/internal/tmux"
)

// MetaHandlers provides HTTP handlers for meta session operations.
type MetaHandlers struct {
	deps *Dependencies
}

// NewMetaHandlers creates a new MetaHandlers with the given dependencies.
func NewMetaHandlers(deps *Dependencies) *MetaHandlers {
	return &MetaHandlers{deps: deps}
}

// MetaSessionResponse is the session details in a meta response.
type MetaSessionResponse struct {
	TmuxSession string    `json:"tmux_session"`
	TmuxWindow  string    `json:"tmux_window"`
	StartedAt   time.Time `json:"started_at"`
	Status      *string   `json:"status,omitempty"`
	Tool        *string   `json:"tool,omitempty"`
	IsOrphaned  bool      `json:"is_orphaned,omitempty"`
}

// MetaStateResponse is the response for GET /meta.
type MetaStateResponse struct {
	State   string               `json:"state"`
	Session *MetaSessionResponse `json:"session,omitempty"`
}

// MetaSpawnResponse is the response for POST /meta/spawn.
type MetaSpawnResponse struct {
	State       string              `json:"state"`
	Session     MetaSessionResponse `json:"session"`
	TmuxSession string              `json:"tmux_session"`
	TmuxWindow  string              `json:"tmux_window"`
}

const metaTmuxSession = "cortex-meta"

// getMetaSession retrieves the meta session from the global meta session store.
func (h *MetaHandlers) getMetaSession() *session.Session {
	if h.deps.MetaSessionManager == nil {
		return nil
	}
	store := h.deps.MetaSessionManager.GetStore()
	sess, _ := store.GetMeta()
	return sess
}

// GetState handles GET /meta - returns meta session state.
func (h *MetaHandlers) GetState(w http.ResponseWriter, r *http.Request) {
	sess := h.getMetaSession()

	stateInfo, err := spawn.DetectMetaState(sess, metaTmuxSession, h.deps.TmuxManager)
	if err != nil {
		h.deps.Logger.Warn("failed to detect meta state", "error", err)
		writeJSON(w, http.StatusOK, MetaStateResponse{State: "normal"})
		return
	}

	resp := MetaStateResponse{
		State: string(stateInfo.State),
	}

	if stateInfo.State != spawn.StateNormal {
		metaResp := &MetaSessionResponse{
			TmuxSession: metaTmuxSession,
			TmuxWindow:  "meta",
		}
		if sess != nil {
			metaResp.StartedAt = sess.StartedAt
			status := string(sess.Status)
			metaResp.Status = &status
			metaResp.Tool = sess.Tool
			metaResp.IsOrphaned = stateInfo.State == spawn.StateOrphaned
		}
		resp.Session = metaResp
	}

	writeJSON(w, http.StatusOK, resp)
}

// Spawn handles POST /meta/spawn - spawns a meta session.
func (h *MetaHandlers) Spawn(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "normal"
	}

	sess := h.getMetaSession()

	// Check tmux is available
	if h.deps.TmuxManager == nil {
		writeError(w, http.StatusServiceUnavailable, "tmux_unavailable", "tmux is not installed")
		return
	}

	// Detect state
	stateInfo, err := spawn.DetectMetaState(sess, metaTmuxSession, h.deps.TmuxManager)
	if err != nil {
		h.deps.Logger.Error("failed to detect meta state", "error", err)
		writeError(w, http.StatusInternalServerError, "state_error", "failed to detect meta session state")
		return
	}

	switch stateInfo.State {
	case spawn.StateActive:
		// Focus the existing window and return 200
		if err := h.deps.TmuxManager.FocusWindow(metaTmuxSession, "meta"); err != nil {
			h.deps.Logger.Warn("failed to focus meta window", "error", err)
		}
		if err := h.deps.TmuxManager.SwitchClient(metaTmuxSession); err != nil {
			h.deps.Logger.Warn("failed to switch tmux client", "session", metaTmuxSession, "error", err)
		}
		metaResp := MetaSessionResponse{
			TmuxSession: metaTmuxSession,
			TmuxWindow:  "meta",
		}
		if sess != nil {
			metaResp.StartedAt = sess.StartedAt
			status := string(sess.Status)
			metaResp.Status = &status
			metaResp.Tool = sess.Tool
		}
		writeJSON(w, http.StatusOK, MetaSpawnResponse{
			State:       "active",
			Session:     metaResp,
			TmuxSession: metaTmuxSession,
			TmuxWindow:  "meta",
		})
		return

	case spawn.StateOrphaned:
		switch mode {
		case "normal":
			writeError(w, http.StatusConflict, "session_orphaned",
				"meta session was orphaned (tmux window closed). Use mode=fresh to start over or mode=resume to continue")
			return
		case "fresh":
			// End old session, then spawn new
			if h.deps.MetaSessionManager != nil {
				store := h.deps.MetaSessionManager.GetStore()
				if endErr := store.EndMeta(); endErr != nil {
					h.deps.Logger.Warn("failed to end orphaned meta session", "error", endErr)
				}
			}
			// Fall through to spawn below
		case "resume":
			// End old session record, spawn with --resume
			if h.deps.MetaSessionManager != nil {
				store := h.deps.MetaSessionManager.GetStore()
				if endErr := store.EndMeta(); endErr != nil {
					h.deps.Logger.Warn("failed to end orphaned meta session for resume", "error", endErr)
				}
			}
			h.spawnMetaSession(w, r, true)
			return
		default:
			writeError(w, http.StatusBadRequest, "invalid_mode", "mode must be 'normal', 'fresh', or 'resume'")
			return
		}

	case spawn.StateNormal:
		if mode != "normal" && mode != "" {
			writeError(w, http.StatusBadRequest, "invalid_mode",
				"cannot use mode '"+mode+"' when no existing meta session exists")
			return
		}
		// Fall through to spawn below
	}

	// Spawn a new meta session
	h.spawnMetaSession(w, r, false)
}

// spawnMetaSession spawns a meta session (new or resumed).
func (h *MetaHandlers) spawnMetaSession(w http.ResponseWriter, r *http.Request, resume bool) {
	var sessStore spawn.SessionStoreInterface
	if h.deps.MetaSessionManager != nil {
		sessStore = h.deps.MetaSessionManager.GetStore()
	}

	spawner := spawn.NewSpawner(spawn.Dependencies{
		TmuxManager:  h.deps.TmuxManager,
		SessionStore: sessStore,
		CortexdPath:  h.deps.CortexdPath,
		Logger:       h.deps.Logger,
	})

	// Load meta agent config from defaults (~/.cortex/defaults/claude-code)
	homeDir, _ := os.UserHomeDir()
	if homeDir == "" {
		homeDir = os.TempDir()
	}

	baseConfigPath := filepath.Join(homeDir, ".cortex", "defaults", "claude-code")
	metaAgent := "claude"
	var metaArgs []string

	var result *spawn.SpawnResult
	var err error

	if resume {
		result, err = spawner.Resume(r.Context(), spawn.ResumeRequest{
			AgentType:   spawn.AgentTypeMeta,
			Agent:       metaAgent,
			TmuxSession: metaTmuxSession,
			WindowName:  "meta",
			AgentArgs:   metaArgs,
		})
	} else {
		result, err = spawner.Spawn(r.Context(), spawn.SpawnRequest{
			AgentType:      spawn.AgentTypeMeta,
			Agent:          metaAgent,
			TmuxSession:    metaTmuxSession,
			AgentArgs:      metaArgs,
			BaseConfigPath: baseConfigPath,
		})
	}

	if err != nil {
		h.deps.Logger.Error("failed to spawn meta", "error", err)
		writeError(w, http.StatusInternalServerError, "spawn_error", "failed to spawn meta session")
		return
	}

	if !result.Success {
		writeError(w, http.StatusInternalServerError, "spawn_error", result.Message)
		return
	}

	writeJSON(w, http.StatusCreated, MetaSpawnResponse{
		State: "active",
		Session: MetaSessionResponse{
			TmuxSession: metaTmuxSession,
			TmuxWindow:  result.TmuxWindow,
		},
		TmuxSession: metaTmuxSession,
		TmuxWindow:  result.TmuxWindow,
	})
}

// Conclude handles POST /meta/conclude - concludes the meta session.
func (h *MetaHandlers) Conclude(w http.ResponseWriter, r *http.Request) {
	var req ConcludeSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "content cannot be empty")
		return
	}

	// Get meta session info before ending
	var tmuxWindow string
	if h.deps.MetaSessionManager != nil {
		store := h.deps.MetaSessionManager.GetStore()
		if sess, err := store.GetMeta(); err == nil && sess != nil {
			tmuxWindow = sess.TmuxWindow
		}

		// End the session
		if endErr := store.EndMeta(); endErr != nil {
			h.deps.Logger.Warn("failed to end meta session", "error", endErr)
		}
	}

	// Kill tmux window if associated (best-effort)
	if tmuxWindow != "" && h.deps.TmuxManager != nil {
		if err := h.deps.TmuxManager.KillWindow(metaTmuxSession, tmuxWindow); err != nil {
			if !tmux.IsWindowNotFound(err) && !tmux.IsSessionNotFound(err) {
				h.deps.Logger.Warn("failed to kill meta tmux window", "error", err)
			}
		}
	}

	writeJSON(w, http.StatusOK, ConcludeSessionResponse{
		Success:  true,
		TicketID: session.MetaSessionKey,
		Message:  "Meta session concluded",
	})
}

// Focus handles POST /meta/focus - focuses the meta tmux window.
func (h *MetaHandlers) Focus(w http.ResponseWriter, r *http.Request) {
	if h.deps.TmuxManager == nil {
		writeError(w, http.StatusServiceUnavailable, "tmux_unavailable", "tmux is not installed")
		return
	}

	if err := h.deps.TmuxManager.FocusWindow(metaTmuxSession, "meta"); err != nil {
		writeError(w, http.StatusInternalServerError, "focus_error", err.Error())
		return
	}

	if err := h.deps.TmuxManager.SwitchClient(metaTmuxSession); err != nil {
		h.deps.Logger.Warn("failed to switch tmux client", "session", metaTmuxSession, "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(FocusResponse{Success: true, Window: "meta"})
}
