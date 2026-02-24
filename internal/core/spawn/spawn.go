package spawn

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/kareemaly/cortex/internal/binpath"
	daemonconfig "github.com/kareemaly/cortex/internal/daemon/config"
	"github.com/kareemaly/cortex/internal/session"
	"github.com/kareemaly/cortex/internal/storage"
	"github.com/kareemaly/cortex/internal/ticket"
)

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
	Create(ticketID, agent, tmuxWindow string) (string, *session.Session, error)
	End(ticketShortID string) error
	GetByTicketID(ticketID string) (*session.Session, error)
	CreateArchitect(agent, tmuxWindow string) (*session.Session, error)
	GetArchitect() (*session.Session, error)
	EndArchitect() error
	CreateCollab(collabID, prompt, agent, tmuxWindow string) (string, *session.Session, error)
}

// TmuxManagerInterface defines the tmux operations needed for spawning.
type TmuxManagerInterface interface {
	WindowExists(session, windowName string) (bool, error)
	SpawnAgent(session, windowName, agentCommand, companionCommand, workingDir, companionWorkingDir string) (int, error)
	SpawnArchitect(session, windowName, agentCommand, companionCommand, workingDir, companionWorkingDir string) error
}

// Dependencies contains the external dependencies for the Spawner.
type Dependencies struct {
	Store             StoreInterface
	SessionStore      SessionStoreInterface
	TmuxManager       TmuxManagerInterface
	Logger            *slog.Logger // optional logger for warnings
	CortexdPath       string       // optional override for cortexd binary path
	MCPConfigDir      string       // optional override for MCP config directory
	SettingsConfigDir string       // optional override for settings config directory
	DefaultsDir       string       // path to defaults (e.g., ~/.cortex/defaults/main) for prompt fallback
}

// Spawner handles spawning agent sessions.
type Spawner struct {
	deps Dependencies
}

// NewSpawner creates a new Spawner with the given dependencies.
func NewSpawner(deps Dependencies) *Spawner {
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
}

// SpawnResult contains the result of a spawn operation.
type SpawnResult struct {
	Success       bool
	TicketID      string
	TmuxWindow    string
	WindowIndex   int
	MCPConfigPath string
	SettingsPath  string
	Message       string
}

// Spawn creates a new agent session.
func (s *Spawner) Spawn(ctx context.Context, req SpawnRequest) (*SpawnResult, error) {
	startedAt := time.Now().UTC().Format(time.RFC3339)
	// Validate request
	if err := s.validateSpawnRequest(req); err != nil {
		return nil, err
	}

	// For ticket agents, check current state
	if req.AgentType == AgentTypeTicketAgent {
		// Look up existing session
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

	// Find cortexd path
	cortexdPath, err := s.getCortexdPath()
	if err != nil {
		return nil, err
	}

	// Generate window name
	windowName := s.generateWindowName(req)

	// Determine working directory based on ticket type
	workingDir, err := getWorkingDirectory(req)
	if err != nil {
		return nil, err
	}

	// Create session in store
	if s.deps.SessionStore != nil {
		switch req.AgentType {
		case AgentTypeTicketAgent:
			_, _, err := s.deps.SessionStore.Create(req.TicketID, req.Agent, windowName)
			if err != nil {
				return nil, err
			}
		case AgentTypeArchitect:
			_, err := s.deps.SessionStore.CreateArchitect(req.Agent, windowName)
			if err != nil {
				return nil, err
			}
		case AgentTypeCollabAgent:
			_, _, err := s.deps.SessionStore.CreateCollab(req.CollabID, req.Prompt, req.Agent, windowName)
			if err != nil {
				return nil, err
			}
		}
	}

	// Generate and write MCP config
	mcpConfig := GenerateMCPConfig(MCPConfigParams{
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

	identifier := req.TicketID
	if identifier == "" && req.CollabID != "" {
		identifier = "collab-" + storage.ShortID(req.CollabID)
	} else if identifier == "" {
		identifier = "architect-" + req.TmuxSession
	}

	mcpConfigPath, err := WriteMCPConfig(mcpConfig, identifier, s.deps.MCPConfigDir)
	if err != nil {
		s.cleanupOnFailure(ctx, req.AgentType, req.TicketID, nil)
		return nil, err
	}

	// Generate and write settings config (hooks) - skip for OpenCode (it doesn't support --settings)
	var settingsPath string
	if req.Agent != "opencode" {
		settingsConfig := GenerateSettingsConfig(SettingsConfigParams{
			CortexdPath:   cortexdPath,
			TicketID:      req.TicketID,
			ArchitectPath: req.ArchitectPath,
		})

		settingsPath, err = WriteSettingsConfig(settingsConfig, identifier, s.deps.SettingsConfigDir)
		if err != nil {
			s.cleanupOnFailure(ctx, req.AgentType, req.TicketID, []string{mcpConfigPath})
			return nil, err
		}
	}

	// Load and build prompt
	pInfo, err := s.buildPrompt(req)
	if err != nil {
		s.cleanupOnFailure(ctx, req.AgentType, req.TicketID, []string{mcpConfigPath, settingsPath})
		return &SpawnResult{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	// Write prompt to temp file
	promptFilePath, err := WritePromptFile(pInfo.PromptText, identifier, "prompt", s.deps.MCPConfigDir)
	if err != nil {
		s.cleanupOnFailure(ctx, req.AgentType, req.TicketID, []string{mcpConfigPath, settingsPath})
		return nil, err
	}

	// Write system prompt to temp file
	var systemPromptFilePath string
	if pInfo.SystemPromptContent != "" {
		systemPromptFilePath, err = WritePromptFile(pInfo.SystemPromptContent, identifier, "sysprompt", s.deps.MCPConfigDir)
		if err != nil {
			s.cleanupOnFailure(ctx, req.AgentType, req.TicketID, []string{mcpConfigPath, settingsPath, promptFilePath})
			return nil, err
		}
	}

	// Generate OpenCode config content if needed
	var openCodeConfigJSON string
	if req.Agent == "opencode" {
		openCodeConfigJSON, err = GenerateOpenCodeConfigContent(mcpConfig, pInfo.SystemPromptContent, req.AgentType, systemPromptFilePath)
		if err != nil {
			s.cleanupOnFailure(ctx, req.AgentType, req.TicketID, nonEmptyStrings(mcpConfigPath, settingsPath, promptFilePath, systemPromptFilePath))
			return nil, err
		}
	}

	// Build launcher params based on agent type
	tempFiles := nonEmptyStrings(mcpConfigPath, settingsPath, promptFilePath, systemPromptFilePath)
	launcherParams := LauncherParams{
		AgentType:            req.Agent,
		PromptFilePath:       promptFilePath,
		SystemPromptFilePath: systemPromptFilePath,
		MCPConfigPath:        mcpConfigPath,
		SettingsPath:         settingsPath,
		AgentArgs:            req.AgentArgs,
		CleanupFiles:         tempFiles,
	}

	switch req.AgentType {
	case AgentTypeArchitect:
		launcherParams.ReplaceSystemPrompt = true
		// For opencode architects, explicitly select the cortex agent
		if req.Agent == "opencode" {
			launcherParams.AgentArgs = append([]string{"--agent", "cortex"}, launcherParams.AgentArgs...)
		}
		launcherParams.EnvVars = map[string]string{
			"CORTEX_TICKET_ID":  session.ArchitectSessionKey,
			"CORTEX_ARCHITECT":  req.ArchitectPath,
			"CORTEX_STARTED_AT": startedAt,
		}
	case AgentTypeTicketAgent:
		launcherParams.EnvVars = map[string]string{
			"CORTEX_TICKET_ID": req.TicketID,
			"CORTEX_ARCHITECT": req.ArchitectPath,
			"CORTEX_TICKET_TYPE": func() string {
				if req.Ticket != nil {
					return req.Ticket.Type
				}
				return ""
			}(),
			"CORTEX_REPO": func() string {
				if req.Ticket != nil {
					return req.Ticket.Repo
				}
				return ""
			}(),
			"CORTEX_STARTED_AT": startedAt,
		}
	case AgentTypeCollabAgent:
		launcherParams.EnvVars = map[string]string{
			"CORTEX_COLLAB_ID":  req.CollabID,
			"CORTEX_ARCHITECT":  req.ArchitectPath,
			"CORTEX_REPO":       req.Repo,
			"CORTEX_STARTED_AT": startedAt,
		}
	}

	if openCodeConfigJSON != "" {
		launcherParams.EnvVars["OPENCODE_CONFIG_CONTENT"] = openCodeConfigJSON
	}

	// Inject OpenCode status plugin for agent status reporting
	if req.Agent == "opencode" {
		pluginContent := GenerateOpenCodeStatusPlugin(
			daemonconfig.DefaultDaemonURL,
			launcherParams.EnvVars["CORTEX_TICKET_ID"],
			req.ArchitectPath,
		)
		pluginDir, pluginErr := WriteOpenCodePluginDir(pluginContent, identifier)
		if pluginErr != nil {
			s.cleanupOnFailure(ctx, req.AgentType, req.TicketID, tempFiles)
			return nil, pluginErr
		}
		launcherParams.CleanupDirs = []string{pluginDir}
		launcherParams.EnvVars["OPENCODE_CONFIG_DIR"] = pluginDir
	}

	launcherPath, err := WriteLauncherScript(launcherParams, identifier, s.deps.MCPConfigDir)
	if err != nil {
		s.cleanupOnFailure(ctx, req.AgentType, req.TicketID, tempFiles)
		return nil, err
	}
	allTempFiles := append(tempFiles, launcherPath)

	// Spawn in tmux
	launchCmd := "bash " + launcherPath
	windowIndex, err := s.spawnInTmux(req, windowName, launchCmd, workingDir)
	if err != nil {
		s.cleanupOnFailure(ctx, req.AgentType, req.TicketID, allTempFiles)
		return &SpawnResult{
			Success: false,
			Message: "failed to spawn agent in tmux: " + err.Error(),
		}, nil
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

	// Find cortexd path
	cortexdPath, err := s.getCortexdPath()
	if err != nil {
		return nil, err
	}

	// Determine identifier for file naming
	identifier := req.TicketID
	if identifier == "" {
		identifier = "architect-" + req.TmuxSession
	}

	// Generate MCP config
	mcpConfig := GenerateMCPConfig(MCPConfigParams{
		CortexdPath:   cortexdPath,
		TicketID:      req.TicketID,
		TicketType:    req.TicketType,
		TicketsDir:    req.TicketsDir,
		ArchitectPath: req.ArchitectPath,
		TmuxSession:   req.TmuxSession,
		StartedAt:     startedAt,
	})

	mcpConfigPath, err := WriteMCPConfig(mcpConfig, identifier, s.deps.MCPConfigDir)
	if err != nil {
		return nil, err
	}

	// Determine env vars based on agent type
	envVars := map[string]string{}
	if req.ArchitectPath != "" {
		envVars["CORTEX_ARCHITECT"] = req.ArchitectPath
	}
	envVars["CORTEX_STARTED_AT"] = startedAt
	switch req.AgentType {
	case AgentTypeTicketAgent:
		envVars["CORTEX_TICKET_ID"] = req.TicketID
		envVars["CORTEX_TICKET_TYPE"] = req.TicketType
	case AgentTypeArchitect:
		envVars["CORTEX_TICKET_ID"] = session.ArchitectSessionKey
	}

	// Generate and write settings config (hooks) - skip for OpenCode (it doesn't support --settings)
	var settingsPath string
	if req.Agent != "opencode" {
		settingsConfig := GenerateSettingsConfig(SettingsConfigParams{
			CortexdPath:   cortexdPath,
			TicketID:      identifier,
			ArchitectPath: req.ArchitectPath,
		})

		settingsPath, err = WriteSettingsConfig(settingsConfig, identifier, s.deps.SettingsConfigDir)
		if err != nil {
			if rmErr := RemoveMCPConfig(mcpConfigPath); rmErr != nil {
				s.logWarn("cleanup: failed to remove MCP config", "path", mcpConfigPath, "error", rmErr)
			}
			return nil, err
		}
	}

	// Generate OpenCode config content for resume (empty system prompt -- resume has no prompts)
	var openCodeConfigJSON string
	if req.Agent == "opencode" {
		openCodeConfigJSON, err = GenerateOpenCodeConfigContent(mcpConfig, "", req.AgentType, "")
		if err != nil {
			if rmErr := RemoveMCPConfig(mcpConfigPath); rmErr != nil {
				s.logWarn("cleanup: failed to remove MCP config", "path", mcpConfigPath, "error", rmErr)
			}
			return nil, err
		}
	}

	// Build launcher script for resume (no prompt files needed)
	tempFiles := nonEmptyStrings(mcpConfigPath, settingsPath)
	launcherParams := LauncherParams{
		AgentType:     req.Agent,
		MCPConfigPath: mcpConfigPath,
		SettingsPath:  settingsPath,
		Resume:        req.SessionID == "",
		ResumeID:      req.SessionID,
		AgentArgs:     req.AgentArgs,
		EnvVars:       envVars,
		CleanupFiles:  tempFiles,
	}

	if req.AgentType == AgentTypeArchitect {
		launcherParams.ReplaceSystemPrompt = true
		// For opencode architects, explicitly select the cortex agent
		if req.Agent == "opencode" {
			launcherParams.AgentArgs = append([]string{"--agent", "cortex"}, launcherParams.AgentArgs...)
		}
	}

	if openCodeConfigJSON != "" {
		launcherParams.EnvVars["OPENCODE_CONFIG_CONTENT"] = openCodeConfigJSON
	}

	// Inject OpenCode status plugin for agent status reporting
	if req.Agent == "opencode" {
		ticketID := envVars["CORTEX_TICKET_ID"]
		pluginContent := GenerateOpenCodeStatusPlugin(
			daemonconfig.DefaultDaemonURL,
			ticketID,
			req.ArchitectPath,
		)
		pluginDir, pluginErr := WriteOpenCodePluginDir(pluginContent, identifier)
		if pluginErr != nil {
			for _, path := range tempFiles {
				if rmErr := os.Remove(path); rmErr != nil && !os.IsNotExist(rmErr) {
					s.logWarn("cleanup: failed to remove temp file", "path", path, "error", rmErr)
				}
			}
			return nil, pluginErr
		}
		launcherParams.CleanupDirs = []string{pluginDir}
		launcherParams.EnvVars["OPENCODE_CONFIG_DIR"] = pluginDir
	}

	launcherPath, err := WriteLauncherScript(launcherParams, identifier, s.deps.MCPConfigDir)
	if err != nil {
		for _, path := range tempFiles {
			if rmErr := os.Remove(path); rmErr != nil && !os.IsNotExist(rmErr) {
				s.logWarn("cleanup: failed to remove temp file", "path", path, "error", rmErr)
			}
		}
		return nil, err
	}
	allTempFiles := append(tempFiles, launcherPath)

	// Create session in store for architect resume
	if s.deps.SessionStore != nil {
		if req.AgentType == AgentTypeArchitect {
			if _, createErr := s.deps.SessionStore.CreateArchitect(req.Agent, req.WindowName); createErr != nil {
				s.logWarn("resume: failed to create architect session", "error", createErr)
			}
		}
	}

	// Spawn in tmux
	launchCmd := "bash " + launcherPath
	var windowIndex int

	// Determine working directory and companion for resume
	workingDir := req.ArchitectPath
	if req.AgentType == AgentTypeTicketAgent && req.TicketType == "work" {
		// For work tickets, we would need to look up the ticket to get the repo path
		// but resume doesn't have access to the full ticket. Use project path for now.
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
		companionCmd := req.Companion
		if req.TicketType == "research" {
			companionCmd = ""
		}
		windowIndex, err = s.deps.TmuxManager.SpawnAgent(req.TmuxSession, req.WindowName, launchCmd, companionCmd, workingDir, workingDir)
	}
	if err != nil {
		for _, path := range allTempFiles {
			if rmErr := os.Remove(path); rmErr != nil && !os.IsNotExist(rmErr) {
				s.logWarn("cleanup: failed to remove temp file", "path", path, "error", rmErr)
			}
		}
		return &SpawnResult{
			Success: false,
			Message: "failed to spawn agent in tmux: " + err.Error(),
		}, nil
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
			existingSess, _ := s.deps.SessionStore.GetByTicketID(req.TicketID)
			if existingSess != nil {
				shortID := storage.ShortID(req.TicketID)
				if err := s.deps.SessionStore.End(shortID); err != nil {
					s.logWarn("fresh: failed to end existing session", "ticketID", req.TicketID, "error", err)
				}
			}
		case AgentTypeArchitect:
			if err := s.deps.SessionStore.EndArchitect(); err != nil {
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
		// Use companion command from config; suppress companion for research tickets
		companionCmd := req.Companion
		if req.Ticket != nil && req.Ticket.Type == "research" {
			companionCmd = ""
		}
		return s.deps.TmuxManager.SpawnAgent(req.TmuxSession, windowName, launchCmd, companionCmd, workingDir, workingDir)
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
				shortID := storage.ShortID(ticketID)
				if err := s.deps.SessionStore.End(shortID); err != nil {
					s.logWarn("cleanup: failed to end session", "ticketID", ticketID, "error", err)
				}
			}
		case AgentTypeArchitect:
			if err := s.deps.SessionStore.EndArchitect(); err != nil {
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

// nonEmptyStrings filters out empty strings from the given values.
func nonEmptyStrings(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, v := range values {
		if v != "" {
			result = append(result, v)
		}
	}
	return result
}
