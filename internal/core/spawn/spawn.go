package spawn

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/hiveryn/agentruntime"
	"github.com/hiveryn/agentruntime/adapter/claude"
	"github.com/hiveryn/agentruntime/adapter/codex"
	"github.com/hiveryn/agentruntime/adapter/opencode"
	"github.com/kareemaly/cortex/internal/binpath"
	"github.com/kareemaly/cortex/internal/core/agent"
	"github.com/kareemaly/cortex/internal/session"
	"github.com/kareemaly/cortex/internal/storage"
	"github.com/kareemaly/cortex/internal/ticket"
)

// newResumeSessionID generates a placeholder UUID when no session store
// is available during resume. Used only for satisfying StartRequest.ID.
var newResumeSessionID = func() string { return uuid.New().String() }

// AgentType represents the type of agent being spawned.
type AgentType string

const (
	// AgentTypeArchitect is the architect agent type.
	AgentTypeArchitect AgentType = "architect"
	// AgentTypeTicketAgent is the ticket agent type.
	AgentTypeTicketAgent AgentType = "ticket_agent"
	// AgentTypeCollabAgent is the collab agent type.
	AgentTypeCollabAgent AgentType = "collab_agent"
)

// StoreInterface defines the ticket store operations needed for spawning.
type StoreInterface interface {
	Get(id string) (*ticket.Ticket, ticket.Status, error)
}

// SessionStoreInterface defines the session store operations needed for spawning.
type SessionStoreInterface interface {
	Create(ticketID, agent, tmuxWindow string) (*session.Session, error)
	EndBySessionID(sessionID string) error
	EndByTicketID(ticketID string) error
	GetByTicketID(ticketID string) (*session.Session, error)
	CreateArchitect(agent, tmuxWindow string) (*session.Session, error)
	GetArchitect() (*session.Session, error)
	EndArchitect() error
	CreateCollab(collabID, prompt, agent, tmuxWindow string) (*session.Session, error)
}

// TmuxManagerInterface defines the tmux operations needed for spawning.
type TmuxManagerInterface interface {
	WindowExists(session, windowName string) (bool, error)
	SpawnAgent(session, windowName, agentCommand, companionCommand, workingDir, companionWorkingDir string) (int, error)
	SpawnArchitect(session, windowName, agentCommand, companionCommand, workingDir, companionWorkingDir string) error
}

// Dependencies contains the external dependencies for the Spawner.
type Dependencies struct {
	Store         StoreInterface
	SessionStore  SessionStoreInterface
	TmuxManager   TmuxManagerInterface
	Logger        *slog.Logger    // optional logger for warnings
	SupervisorCtx context.Context // daemon-root context for long-lived agent supervisors; nil → context.Background
	CortexdPath   string          // optional override for cortexd binary path
	MCPConfigDir  string          // optional override for MCP config directory
	DefaultsDir   string          // path to defaults (e.g., ~/.cortex/defaults/main) for prompt fallback

	// HubEventSource, when non-nil, supplies per-session Hub event streams to
	// the supervisor. The supervisor forwards these to /agent/status → SSE.
	HubEventSource func(ctx context.Context, sessionID string) <-chan agent.HubEvent
}

// Spawner handles spawning agent sessions.
type Spawner struct {
	deps Dependencies
}

// NewSpawner creates a new Spawner with the given dependencies.
func NewSpawner(deps Dependencies) *Spawner {
	if deps.SupervisorCtx == nil {
		deps.SupervisorCtx = context.Background()
	}
	return &Spawner{deps: deps}
}

// logWarn logs a warning message if a logger is configured.
func (s *Spawner) logWarn(msg string, args ...any) {
	if s.deps.Logger != nil {
		s.deps.Logger.Warn(msg, args...)
	}
}

// SpawnRequest contains parameters for spawning a new agent session.
type SpawnRequest struct {
	AgentType     AgentType
	Agent         string // agent identifier (e.g., "claude")
	TmuxSession   string
	ArchitectPath string
	TicketsDir    string

	// For ticket agents
	TicketID string
	Ticket   *ticket.Ticket

	// For architect agents
	ArchitectName string

	// For collab agents
	CollabID string // unique UUID for this collab session
	Prompt   string // kickoff prompt text
	Repo     string // working directory (used for collab agents)

	// Companion pane command (from cortex.yaml)
	Companion string

	// Extra CLI args appended to the agent command
	AgentArgs []string
	// Per-variant env vars merged on top of system env (variant wins on conflict)
	EnvVars map[string]string
}

// ResumeRequest contains parameters for resuming an orphaned session.
type ResumeRequest struct {
	AgentType     AgentType
	Agent         string // agent identifier (e.g., "claude", "opencode")
	TmuxSession   string
	ArchitectPath string
	TicketsDir    string
	SessionID     string
	WindowName    string

	// For ticket agents
	TicketID   string
	TicketType string // ticket type
	Companion  string // companion pane command (from cortex.yaml)

	// Extra CLI args appended to the agent command
	AgentArgs []string
	// Per-variant env vars merged on top of system env (variant wins on conflict)
	EnvVars map[string]string
}

// SpawnResult contains the result of a spawn operation.
type SpawnResult struct {
	Success       bool
	TicketID      string
	TmuxWindow    string
	WindowIndex   int
	MCPConfigPath string
	Message       string
}

// Spawn creates a new agent session.
func (s *Spawner) Spawn(ctx context.Context, req SpawnRequest) (*SpawnResult, error) {
	startedAt := time.Now().UTC().Format(time.RFC3339)
	if err := s.validateSpawnRequest(req); err != nil {
		return nil, err
	}

	if req.AgentType == AgentTypeTicketAgent {
		var existingSess *session.Session
		if s.deps.SessionStore != nil {
			existingSess, _ = s.deps.SessionStore.GetByTicketID(req.TicketID)
		}
		stateInfo, err := DetectTicketState(existingSess, req.TmuxSession, s.deps.TmuxManager)
		if err != nil {
			return nil, err
		}
		if stateInfo.State == StateActive {
			return &SpawnResult{
				Success:  false,
				TicketID: req.TicketID,
				Message:  "ticket already has an active session",
			}, nil
		}
	}

	cortexdPath, err := s.getCortexdPath()
	if err != nil {
		return nil, err
	}

	windowName := s.generateWindowName(req)
	workingDir, err := getWorkingDirectory(req)
	if err != nil {
		return nil, err
	}

	var sessionIDForStatus string
	if s.deps.SessionStore != nil {
		switch req.AgentType {
		case AgentTypeTicketAgent:
			sess, err := s.deps.SessionStore.Create(req.TicketID, req.Agent, windowName)
			if err != nil {
				return nil, err
			}
			if sess != nil {
				sessionIDForStatus = sess.SessionID
			}
		case AgentTypeArchitect:
			sess, err := s.deps.SessionStore.CreateArchitect(req.Agent, windowName)
			if err != nil {
				return nil, err
			}
			if sess != nil {
				sessionIDForStatus = sess.SessionID
			}
		case AgentTypeCollabAgent:
			sess, err := s.deps.SessionStore.CreateCollab(req.CollabID, req.Prompt, req.Agent, windowName)
			if err != nil {
				return nil, err
			}
			if sess != nil {
				sessionIDForStatus = sess.SessionID
			}
		}
	}

	mcpServerConfig := BuildMCPServerConfig(MCPConfigParams{
		CortexdPath: cortexdPath,
		TicketID:    req.TicketID,
		TicketType: func() string {
			if req.Ticket != nil {
				return req.Ticket.Type
			}
			return ""
		}(),
		TicketsDir:    req.TicketsDir,
		ArchitectPath: req.ArchitectPath,
		TmuxSession:   req.TmuxSession,
		StartedAt:     startedAt,
		CollabID:      req.CollabID,
	})

	pInfo, err := s.buildPrompt(req)
	if err != nil {
		s.cleanupOnFailure(ctx, req.AgentType, req.TicketID, nil)
		return &SpawnResult{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	identifier := req.TicketID
	if identifier == "" && req.CollabID != "" {
		identifier = "collab-" + storage.ShortID(req.CollabID)
	} else if identifier == "" {
		identifier = "architect-" + req.TmuxSession
	}

	adapter, err := s.adapterFor(req.Agent, req.AgentType)
	if err != nil {
		s.cleanupOnFailure(ctx, req.AgentType, req.TicketID, nil)
		return nil, err
	}

	startReq := agentruntime.StartRequest{
		ID:           sessionIDForStatus,
		Agent:        s.agentKind(req.Agent),
		Args:         req.AgentArgs,
		Workdir:      workingDir,
		Prompt:       pInfo.PromptText,
		Instructions: pInfo.SystemPromptContent,
		MCPServers:   []agentruntime.MCPServerConfig{mcpServerConfig},
	}

	if req.AgentType == AgentTypeArchitect {
		if req.Agent == "opencode" {
			if !hasAgentFlag(startReq.Args) {
				startReq.Args = append([]string{"--agent", "cortex"}, startReq.Args...)
			}
			startReq.OpenCodeAgentConfig = map[string]agentruntime.OpenCodeAgentConfig{
				"cortex": {
					Description: "Cortex architect agent",
					Mode:        "primary",
					Prompt:      pInfo.SystemPromptContent,
					Permission:  map[string]string{"*": "allow"},
				},
			}
		}
	}

	cortexEnv := s.buildCortexEnv(req, startedAt)
	startReq.Env = mergeEnvMaps(req.EnvVars, cortexEnv)

	spec, err := adapter.PrepareLaunch(ctx, startReq)
	if err != nil {
		s.cleanupOnFailure(ctx, req.AgentType, req.TicketID, nil)
		return &SpawnResult{
			Success: false,
			Message: "failed to prepare agent launch: " + err.Error(),
		}, nil
	}

	launcherPath, err := WriteLauncherScript(spec, cortexEnv, identifier, s.deps.MCPConfigDir)
	if err != nil {
		s.cleanupOnFailure(ctx, req.AgentType, req.TicketID, nil)
		return nil, err
	}

	// Spawn in tmux
	launchCmd := "bash " + launcherPath
	allCleanupFiles := append([]string{launcherPath}, spec.CleanupPaths...)

	windowIndex, err := s.spawnInTmux(req, windowName, launchCmd, workingDir)
	if err != nil {
		s.cleanupOnFailure(ctx, req.AgentType, req.TicketID, allCleanupFiles)
		return &SpawnResult{
			Success: false,
			Message: "failed to spawn agent in tmux: " + err.Error(),
		}, nil
	}

	livenessPath := launcherPath
	if len(spec.CleanupPaths) > 0 {
		livenessPath = spec.CleanupPaths[0]
	}

	ticketIDForStatus := cortexEnv["CORTEX_TICKET_ID"]
	supCtx := s.deps.SupervisorCtx
	supParams := agentSupervisorParams{
		SessionID:      sessionIDForStatus,
		TicketID:       ticketIDForStatus,
		ArchitectPath:  req.ArchitectPath,
		LivenessPath:   livenessPath,
		HubEventSource: s.deps.HubEventSource,
		Logger:         s.deps.Logger,
	}
	if _, err := startAgentSupervisor(supCtx, supParams); err != nil {
		s.logWarn("failed to start agent supervisor", "agent", req.Agent, "error", err)
	}

	return &SpawnResult{
		Success:     true,
		TmuxWindow:  windowName,
		WindowIndex: windowIndex,
		Message:     "Agent session spawned in tmux window '" + windowName + "'",
	}, nil
}

// Resume resumes an orphaned agent session (ticket or architect).
func (s *Spawner) Resume(ctx context.Context, req ResumeRequest) (*SpawnResult, error) {
	if req.AgentType == AgentTypeTicketAgent && req.TicketID == "" {
		return nil, &ConfigError{Field: "TicketID", Message: "cannot be empty for ticket agent resume"}
	}

	startedAt := time.Now().UTC().Format(time.RFC3339)

	cortexdPath, err := s.getCortexdPath()
	if err != nil {
		return nil, err
	}

	identifier := req.TicketID
	if identifier == "" {
		identifier = "architect-" + req.TmuxSession
	}

	mcpServerConfig := BuildMCPServerConfig(MCPConfigParams{
		CortexdPath:   cortexdPath,
		TicketID:      req.TicketID,
		TicketType:    req.TicketType,
		TicketsDir:    req.TicketsDir,
		ArchitectPath: req.ArchitectPath,
		TmuxSession:   req.TmuxSession,
		StartedAt:     startedAt,
	})

	adapter, err := s.adapterFor(req.Agent, req.AgentType)
	if err != nil {
		return nil, err
	}

	cortexEnv := map[string]string{
		"CORTEX_STARTED_AT": startedAt,
	}
	switch req.AgentType {
	case AgentTypeTicketAgent:
		cortexEnv["CORTEX_TICKET_ID"] = req.TicketID
		cortexEnv["CORTEX_TICKET_TYPE"] = req.TicketType
	case AgentTypeArchitect:
		cortexEnv["CORTEX_TICKET_ID"] = session.ArchitectSessionKey
	}

	// Resolve the session ID first so it can be used as StartRequest.ID.
	var resumeSessionID string
	if s.deps.SessionStore != nil {
		switch req.AgentType {
		case AgentTypeArchitect:
			if sess, createErr := s.deps.SessionStore.CreateArchitect(req.Agent, req.WindowName); createErr != nil {
				s.logWarn("resume: failed to create architect session", "error", createErr)
			} else if sess != nil {
				resumeSessionID = sess.SessionID
			}
		case AgentTypeTicketAgent:
			if existing, _ := s.deps.SessionStore.GetByTicketID(req.TicketID); existing != nil {
				resumeSessionID = existing.SessionID
			}
		}
	}
	if resumeSessionID == "" {
		resumeSessionID = newResumeSessionID()
	}

	startReq := agentruntime.StartRequest{
		ID:         resumeSessionID,
		Agent:      s.agentKind(req.Agent),
		Args:       req.AgentArgs,
		Workdir:    req.ArchitectPath,
		MCPServers: []agentruntime.MCPServerConfig{mcpServerConfig},
		Resume:     req.SessionID == "",
		ResumeID:   req.SessionID,
		Env:        mergeEnvMaps(cortexEnv, req.EnvVars),
	}

	if req.AgentType == AgentTypeArchitect {
		if req.Agent == "opencode" {
			if !hasAgentFlag(startReq.Args) {
				startReq.Args = append([]string{"--agent", "cortex"}, startReq.Args...)
			}
		}
	}

	spec, err := adapter.PrepareLaunch(ctx, startReq)
	if err != nil {
		return nil, err
	}

	launcherPath, err := WriteLauncherScript(spec, cortexEnv, identifier, s.deps.MCPConfigDir)
	if err != nil {
		return nil, err
	}

	launchCmd := "bash " + launcherPath
	var windowIndex int

	workingDir := req.ArchitectPath
	if req.AgentType == AgentTypeTicketAgent && req.TicketType == "work" {
		workingDir = req.ArchitectPath
	}

	switch req.AgentType {
	case AgentTypeArchitect:
		companionCmd := req.Companion
		if companionCmd == "" {
			companionCmd = "cortex architect show"
		}
		err = s.deps.TmuxManager.SpawnArchitect(req.TmuxSession, req.WindowName, launchCmd, companionCmd, workingDir, workingDir)
	case AgentTypeTicketAgent:
		windowIndex, err = s.deps.TmuxManager.SpawnAgent(req.TmuxSession, req.WindowName, launchCmd, req.Companion, workingDir, workingDir)
	}
	if err != nil {
		{
			for _, path := range spec.CleanupPaths {
				if rmErr := os.Remove(path); rmErr != nil && !os.IsNotExist(rmErr) {
					s.logWarn("cleanup: failed to remove temp file", "path", path, "error", rmErr)
				}
			}
			_ = os.Remove(launcherPath)
		}
		return &SpawnResult{
			Success: false,
			Message: "failed to spawn agent in tmux: " + err.Error(),
		}, nil
	}

	livenessPath := launcherPath
	if len(spec.CleanupPaths) > 0 {
		livenessPath = spec.CleanupPaths[0]
	}

	ticketIDForStatus := cortexEnv["CORTEX_TICKET_ID"]
	supCtx := s.deps.SupervisorCtx
	supParams := agentSupervisorParams{
		SessionID:      resumeSessionID,
		TicketID:       ticketIDForStatus,
		ArchitectPath:  req.ArchitectPath,
		LivenessPath:   livenessPath,
		HubEventSource: s.deps.HubEventSource,
		Logger:         s.deps.Logger,
	}
	if _, err := startAgentSupervisor(supCtx, supParams); err != nil {
		s.logWarn("resume: supervisor start failed", "agent", req.Agent, "error", err)
	}

	return &SpawnResult{
		Success:     true,
		TmuxWindow:  req.WindowName,
		WindowIndex: windowIndex,
		Message:     "Session resumed in tmux window '" + req.WindowName + "'",
	}, nil
}

// Fresh clears any existing session and spawns a new one.
func (s *Spawner) Fresh(ctx context.Context, req SpawnRequest) (*SpawnResult, error) {
	if s.deps.SessionStore != nil {
		switch req.AgentType {
		case AgentTypeTicketAgent:
			if err := s.deps.SessionStore.EndByTicketID(req.TicketID); err != nil && !storage.IsNotFound(err) {
				s.logWarn("fresh: failed to end existing session", "ticketID", req.TicketID, "error", err)
			}
		case AgentTypeArchitect:
			if err := s.deps.SessionStore.EndArchitect(); err != nil && !storage.IsNotFound(err) {
				s.logWarn("fresh: failed to end existing architect session", "error", err)
			}
		}
	}

	return s.Spawn(ctx, req)
}

// getCortexdPath returns the path to cortexd binary.
func (s *Spawner) getCortexdPath() (string, error) {
	if s.deps.CortexdPath != "" {
		return s.deps.CortexdPath, nil
	}
	path, err := binpath.FindCortexd()
	if err != nil {
		return "", &BinaryNotFoundError{Binary: "cortexd", Cause: err}
	}
	return path, nil
}

// spawnInTmux spawns the agent in a tmux window.
func (s *Spawner) spawnInTmux(req SpawnRequest, windowName, launchCmd, workingDir string) (int, error) {
	switch req.AgentType {
	case AgentTypeTicketAgent:
		return s.deps.TmuxManager.SpawnAgent(req.TmuxSession, windowName, launchCmd, req.Companion, workingDir, workingDir)
	case AgentTypeArchitect:
		// Use companion command from config, or default to showing kanban board
		companionCmd := req.Companion
		if companionCmd == "" {
			companionCmd = "cortex architect show"
		}
		err := s.deps.TmuxManager.SpawnArchitect(req.TmuxSession, windowName, launchCmd, companionCmd, workingDir, req.ArchitectPath)
		return 0, err
	case AgentTypeCollabAgent:
		// Collab agents spawn as simple agent windows (no companion)
		return s.deps.TmuxManager.SpawnAgent(req.TmuxSession, windowName, launchCmd, req.Companion, workingDir, workingDir)
	default:
		return 0, &ConfigError{Field: "AgentType", Message: "unknown agent type: " + string(req.AgentType)}
	}
}

// cleanupOnFailure cleans up resources when spawn fails.
func (s *Spawner) cleanupOnFailure(_ context.Context, agentType AgentType, ticketID string, tempFiles []string) {
	if s.deps.SessionStore != nil {
		switch agentType {
		case AgentTypeTicketAgent:
			if ticketID != "" {
				if err := s.deps.SessionStore.EndByTicketID(ticketID); err != nil && !storage.IsNotFound(err) {
					s.logWarn("cleanup: failed to end session", "ticketID", ticketID, "error", err)
				}
			}
		case AgentTypeArchitect:
			if err := s.deps.SessionStore.EndArchitect(); err != nil && !storage.IsNotFound(err) {
				s.logWarn("cleanup: failed to end architect session", "error", err)
			}
			// AgentTypeCollabAgent cleanup handled in SpawnCollab via EndCollab
		}
	}
	for _, path := range tempFiles {
		if path == "" {
			continue
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			s.logWarn("cleanup: failed to remove temp file", "path", path, "error", err)
		}
	}
}

// adapterFor returns an agentruntime adapter for the given agent string.
// Architect sessions use --system-prompt (full replace); ticket/collab use
// --append-system-prompt (additive) for Claude.
func (s *Spawner) adapterFor(agent string, agentType AgentType) (agentruntime.Adapter, error) {
	switch agent {
	case "claude":
		opts := claude.DefaultOptions()
		opts.AppendInstructions = agentType != AgentTypeArchitect
		return claude.New(opts), nil
	case "codex":
		return codex.New(codex.DefaultOptions()), nil
	case "opencode":
		return opencode.New(opencode.DefaultOptions()), nil
	default:
		return nil, fmt.Errorf("unknown agent: %s", agent)
	}
}

// agentKind maps a cortex agent string to an agentruntime AgentKind.
func (s *Spawner) agentKind(agent string) agentruntime.AgentKind {
	switch agent {
	case "claude":
		return agentruntime.AgentClaude
	case "codex":
		return agentruntime.AgentCodex
	case "opencode":
		return agentruntime.AgentOpenCode
	default:
		return agentruntime.AgentKind(agent)
	}
}

// buildCortexEnv returns the per-session agent env vars set by Cortex.
func (s *Spawner) buildCortexEnv(req SpawnRequest, startedAt string) map[string]string {
	env := map[string]string{
		"CORTEX_STARTED_AT": startedAt,
	}
	switch req.AgentType {
	case AgentTypeArchitect:
		env["CORTEX_TICKET_ID"] = session.ArchitectSessionKey
	case AgentTypeTicketAgent:
		env["CORTEX_TICKET_ID"] = req.TicketID
		if req.Ticket != nil {
			env["CORTEX_TICKET_TYPE"] = req.Ticket.Type
			env["CORTEX_REPO"] = req.Ticket.Repo
		}
	case AgentTypeCollabAgent:
		env["CORTEX_COLLAB_ID"] = req.CollabID
		env["CORTEX_REPO"] = req.Repo
	}
	return env
}

func hasAgentFlag(args []string) bool {
	for _, a := range args {
		if a == "--agent" {
			return true
		}
	}
	return false
}

// mergeEnvMaps merges cortex env with variant env, where variant wins.
func mergeEnvMaps(cortexEnv, variantEnv map[string]string) map[string]string {
	out := make(map[string]string, len(cortexEnv)+len(variantEnv))
	for k, v := range cortexEnv {
		out[k] = v
	}
	for k, v := range variantEnv {
		out[k] = v
	}
	return out
}
