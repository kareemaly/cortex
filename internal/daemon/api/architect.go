package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	architectconfig "github.com/kareemaly/cortex/internal/architect/config"
	"github.com/kareemaly/cortex/internal/architectsession"
	"github.com/kareemaly/cortex/internal/core/spawn"
	"github.com/kareemaly/cortex/internal/events"
	"github.com/kareemaly/cortex/internal/session"
)

type ArchitectHandlers struct {
	deps *Dependencies
}

func NewArchitectHandlers(deps *Dependencies) *ArchitectHandlers {
	return &ArchitectHandlers{deps: deps}
}

func (h *ArchitectHandlers) getSessionAndConfig(projectPath string) (sessionName string, sess *session.Session, projectCfg *architectconfig.Config, err error) {
	projectCfg, err = mergeProjectConfig(projectPath)
	if err != nil {
		return "", nil, nil, err
	}

	sessionName = projectCfg.GetTmuxSessionName()

	if h.deps.SessionManager != nil {
		sessStore := h.deps.SessionManager.GetStore(projectPath)
		sess, _ = sessStore.GetArchitect()
	}

	return sessionName, sess, projectCfg, nil
}

func (h *ArchitectHandlers) GetState(w http.ResponseWriter, r *http.Request) {
	projectPath := GetArchitectPath(r.Context())

	sessionName, sess, _, err := h.getSessionAndConfig(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "config_error", "failed to load project config")
		return
	}

	stateInfo, err := spawn.DetectArchitectState(sess, sessionName, h.deps.TmuxManager)
	if err != nil {
		h.deps.Logger.Warn("failed to detect architect state", "error", err)
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
			archResp.ID = sess.SessionID
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

func (h *ArchitectHandlers) Spawn(w http.ResponseWriter, r *http.Request) {
	projectPath := GetArchitectPath(r.Context())
	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "normal"
	}
	variantName := r.URL.Query().Get("variant")

	sessionName, sess, projectCfg, err := h.getSessionAndConfig(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "config_error", "failed to load project config")
		return
	}

	if h.deps.TmuxManager == nil {
		writeError(w, http.StatusServiceUnavailable, "tmux_unavailable", "tmux is not installed")
		return
	}

	stateInfo, err := spawn.DetectArchitectState(sess, sessionName, h.deps.TmuxManager)
	if err != nil {
		h.deps.Logger.Error("failed to detect architect state", "error", err)
		writeError(w, http.StatusInternalServerError, "state_error", "failed to detect architect session state")
		return
	}

	if stateInfo.State == spawn.StateActive {
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
			archResp.ID = sess.SessionID
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
	}

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
	av, err := projectCfg.ResolveVariant(variantName)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_variant", err.Error())
		return
	}

	agent := string(av.Agent)
	if agent == "" {
		agent = "claude"
	}

	switch stateInfo.State {
	case spawn.StateOrphaned:
		switch mode {
		case "normal":
			writeError(w, http.StatusConflict, "session_orphaned",
				"session orphaned — use --mode=resume to continue or --mode=fresh to restart")
			return
		case "fresh":
			if h.deps.SessionManager != nil {
				sessStore := h.deps.SessionManager.GetStore(projectPath)
				if endErr := sessStore.EndArchitect(); endErr != nil {
					h.deps.Logger.Warn("failed to end orphaned architect session", "error", endErr)
				}
			}
		case "resume":
			if h.deps.SessionManager != nil {
				sessStore := h.deps.SessionManager.GetStore(projectPath)
				if endErr := sessStore.EndArchitect(); endErr != nil {
					h.deps.Logger.Warn("failed to end orphaned architect session for resume", "error", endErr)
				}
			}
			h.spawnArchitectSession(w, r, projectPath, sessionName, projectCfg, agent, av.Args, av.Env, "cortex architect show", true)
			return
		default:
			writeError(w, http.StatusBadRequest, "invalid_mode", "mode must be 'normal', 'fresh', or 'resume'")
			return
		}

	case spawn.StateNormal:
		if mode != "normal" && mode != "" {
			writeError(w, http.StatusBadRequest, "invalid_mode",
				"no existing session — --mode="+mode+" requires an active or orphaned session")
			return
		}
	}

	h.spawnArchitectSession(w, r, projectPath, sessionName, projectCfg, agent, av.Args, av.Env, "cortex architect show", false)
}

func (h *ArchitectHandlers) spawnArchitectSession(w http.ResponseWriter, r *http.Request, projectPath, sessionName string, projectCfg *architectconfig.Config, agent string, agentArgs []string, agentEnv map[string]string, companion string, resume bool) {
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

	var result *spawn.SpawnResult
	var err error

	if resume {
		result, err = spawner.Resume(r.Context(), spawn.ResumeRequest{
			AgentType:     spawn.AgentTypeArchitect,
			Agent:         agent,
			TmuxSession:   sessionName,
			ArchitectPath: projectPath,
			TicketsDir:    ticketsDir,
			WindowName:    "architect",
			Companion:     companion,
			AgentArgs:     agentArgs,
			EnvVars:       agentEnv,
		})
	} else {
		result, err = spawner.Spawn(r.Context(), spawn.SpawnRequest{
			AgentType:     spawn.AgentTypeArchitect,
			Agent:         agent,
			TmuxSession:   sessionName,
			ArchitectPath: projectPath,
			TicketsDir:    ticketsDir,
			ArchitectName: sessionName,
			Companion:     companion,
			AgentArgs:     agentArgs,
			EnvVars:       agentEnv,
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

	h.deps.Bus.Emit(events.Event{
		Type:          events.SessionStarted,
		ArchitectPath: projectPath,
	})

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

func (h *ArchitectHandlers) Conclude(w http.ResponseWriter, r *http.Request) {
	projectPath := GetArchitectPath(r.Context())

	var req ConcludeSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "content cannot be empty")
		return
	}

	var archSessionID string
	var tmuxWindow string
	var agent string
	if h.deps.SessionManager != nil {
		sessStore := h.deps.SessionManager.GetStore(projectPath)
		if sess, err := sessStore.GetArchitect(); err == nil && sess != nil {
			archSessionID = sess.SessionID
			tmuxWindow = sess.TmuxWindow
			agent = sess.Agent
		}

		if endErr := sessStore.EndArchitect(); endErr != nil {
			h.deps.Logger.Warn("failed to end architect session", "error", endErr)
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
	concMeta := architectsession.ConclusionMeta{
		StartedAt:   startedAt,
		ConcludedAt: concludedAt,
		Agent:       agent,
		Profile:     "",
	}

	if err := architectsession.EnsureDir(projectPath); err != nil {
		h.deps.Logger.Warn("failed to ensure architect-sessions dir", "error", err)
	}
	if archSessionID != "" {
		if writeErr := architectsession.WriteConclusion(projectPath, archSessionID, concMeta, req.Content); writeErr != nil {
			h.deps.Logger.Warn("failed to write architect conclusion", "error", writeErr)
		}
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
		TicketID: session.ArchitectSessionKey,
		Message:  "Architect session concluded",
	})
}

func (h *ArchitectHandlers) Focus(w http.ResponseWriter, r *http.Request) {
	projectPath := GetArchitectPath(r.Context())

	projectCfg, err := architectconfig.Load(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "config_error", "failed to load project config")
		return
	}

	sessionName := projectCfg.GetTmuxSessionName()

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
