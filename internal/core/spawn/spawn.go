package spawn

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kareemaly/cortex/internal/binpath"
	"github.com/kareemaly/cortex/internal/cli/sdk"
	daemonconfig "github.com/kareemaly/cortex/internal/daemon/config"
	"github.com/kareemaly/cortex/internal/prompt"
	"github.com/kareemaly/cortex/internal/session"
	"github.com/kareemaly/cortex/internal/storage"
	"github.com/kareemaly/cortex/internal/ticket"
	"github.com/kareemaly/cortex/internal/worktree"
)

// generateSessionID generates a unique session ID for worktrees.
func generateSessionID() string {
	return uuid.New().String()
}

// AgentType represents the type of agent being spawned.
type AgentType string

const (
	// AgentTypeArchitect is the architect agent type.
	AgentTypeArchitect AgentType = "architect"
	// AgentTypeTicketAgent is the ticket agent type.
	AgentTypeTicketAgent AgentType = "ticket_agent"
	// AgentTypeMeta is the meta agent type (global, above architects).
	AgentTypeMeta AgentType = "meta"
)

// StoreInterface defines the ticket store operations needed for spawning.
type StoreInterface interface {
	Get(id string) (*ticket.Ticket, ticket.Status, error)
}

// SessionStoreInterface defines the session store operations needed for spawning.
type SessionStoreInterface interface {
	Create(ticketID, agent, tmuxWindow string, worktreePath, featureBranch *string) (string, *session.Session, error)
	End(ticketShortID string) error
	GetByTicketID(ticketID string) (*session.Session, error)
	CreateArchitect(agent, tmuxWindow string) (*session.Session, error)
	GetArchitect() (*session.Session, error)
	EndArchitect() error
	CreateMeta(agent, tmuxWindow string) (*session.Session, error)
	GetMeta() (*session.Session, error)
	EndMeta() error
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
	AgentType   AgentType
	Agent       string // agent identifier (e.g., "claude")
	TmuxSession string
	ProjectPath string
	TicketsDir  string

	// For ticket agents
	TicketID    string
	Ticket      *ticket.Ticket
	UseWorktree bool // if true, spawn in a git worktree

	// For architect agents
	ProjectName string

	// Extra CLI args appended to the agent command
	AgentArgs []string

	// BaseConfigPath is the resolved extend path from project config.
	// Used for prompt fallback resolution.
	BaseConfigPath string
}

// ResumeRequest contains parameters for resuming an orphaned session.
type ResumeRequest struct {
	AgentType   AgentType
	Agent       string // agent identifier (e.g., "claude", "opencode")
	TmuxSession string
	ProjectPath string
	TicketsDir  string
	SessionID   string
	WindowName  string

	// For ticket agents
	TicketID   string
	TicketType string // ticket type (work/debug/research)

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

	// Determine working directory and worktree info
	workingDir := req.ProjectPath
	var worktreePath, featureBranch *string

	if req.UseWorktree && req.AgentType == AgentTypeTicketAgent {
		wm := worktree.NewManager(req.ProjectPath)
		slug := storage.GenerateSlug(req.Ticket.Title, "ticket")

		// Generate session ID for worktree path
		sessionID := generateSessionID()

		wtPath, branch, err := wm.Create(ctx, sessionID, slug)
		if err != nil {
			return nil, fmt.Errorf("create worktree: %w", err)
		}

		worktreePath = &wtPath
		featureBranch = &branch
		workingDir = wtPath
	}

	// Create session in store
	if s.deps.SessionStore != nil {
		switch req.AgentType {
		case AgentTypeTicketAgent:
			_, _, err := s.deps.SessionStore.Create(req.TicketID, req.Agent, windowName, worktreePath, featureBranch)
			if err != nil {
				if worktreePath != nil && featureBranch != nil {
					wm := worktree.NewManager(req.ProjectPath)
					_ = wm.Remove(ctx, *worktreePath, *featureBranch)
				}
				return nil, err
			}
		case AgentTypeArchitect:
			_, err := s.deps.SessionStore.CreateArchitect(req.Agent, windowName)
			if err != nil {
				return nil, err
			}
		case AgentTypeMeta:
			_, err := s.deps.SessionStore.CreateMeta(req.Agent, windowName)
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
		TicketsDir:  req.TicketsDir,
		ProjectPath: req.ProjectPath,
		TmuxSession: req.TmuxSession,
		IsMeta:      req.AgentType == AgentTypeMeta,
	})

	identifier := req.TicketID
	if identifier == "" {
		if req.AgentType == AgentTypeMeta {
			identifier = "meta"
		} else {
			identifier = "architect-" + req.TmuxSession
		}
	}

	mcpConfigPath, err := WriteMCPConfig(mcpConfig, identifier, s.deps.MCPConfigDir)
	if err != nil {
		s.cleanupOnFailure(ctx, req.AgentType, req.TicketID, nil, worktreePath, featureBranch, req.ProjectPath)
		return nil, err
	}

	// Generate and write settings config (hooks) - skip for OpenCode (it doesn't support --settings)
	var settingsPath string
	if req.Agent != "opencode" {
		settingsConfig := GenerateSettingsConfig(SettingsConfigParams{
			CortexdPath: cortexdPath,
			TicketID:    req.TicketID,
			ProjectPath: req.ProjectPath,
		})

		settingsPath, err = WriteSettingsConfig(settingsConfig, identifier, s.deps.SettingsConfigDir)
		if err != nil {
			s.cleanupOnFailure(ctx, req.AgentType, req.TicketID, []string{mcpConfigPath}, worktreePath, featureBranch, req.ProjectPath)
			return nil, err
		}
	}

	// Load and build prompt
	pInfo, err := s.buildPrompt(req, worktreePath, featureBranch)
	if err != nil {
		s.cleanupOnFailure(ctx, req.AgentType, req.TicketID, []string{mcpConfigPath, settingsPath}, worktreePath, featureBranch, req.ProjectPath)
		return &SpawnResult{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	// Write prompt to temp file
	promptFilePath, err := WritePromptFile(pInfo.PromptText, identifier, "prompt", s.deps.MCPConfigDir)
	if err != nil {
		s.cleanupOnFailure(ctx, req.AgentType, req.TicketID, []string{mcpConfigPath, settingsPath}, worktreePath, featureBranch, req.ProjectPath)
		return nil, err
	}

	// Write system prompt to temp file
	var systemPromptFilePath string
	if pInfo.SystemPromptContent != "" {
		systemPromptFilePath, err = WritePromptFile(pInfo.SystemPromptContent, identifier, "sysprompt", s.deps.MCPConfigDir)
		if err != nil {
			s.cleanupOnFailure(ctx, req.AgentType, req.TicketID, []string{mcpConfigPath, settingsPath, promptFilePath}, worktreePath, featureBranch, req.ProjectPath)
			return nil, err
		}
	}

	// Generate OpenCode config content if needed
	var openCodeConfigJSON string
	if req.Agent == "opencode" {
		openCodeConfigJSON, err = GenerateOpenCodeConfigContent(mcpConfig, pInfo.SystemPromptContent)
		if err != nil {
			s.cleanupOnFailure(ctx, req.AgentType, req.TicketID, nonEmptyStrings(mcpConfigPath, settingsPath, promptFilePath, systemPromptFilePath), worktreePath, featureBranch, req.ProjectPath)
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
		launcherParams.EnvVars = map[string]string{
			"CORTEX_TICKET_ID": session.ArchitectSessionKey,
			"CORTEX_PROJECT":   req.ProjectPath,
		}
	case AgentTypeTicketAgent:
		launcherParams.EnvVars = map[string]string{
			"CORTEX_TICKET_ID": req.TicketID,
			"CORTEX_PROJECT":   req.ProjectPath,
		}
	case AgentTypeMeta:
		launcherParams.ReplaceSystemPrompt = true
		launcherParams.EnvVars = map[string]string{
			"CORTEX_TICKET_ID": session.MetaSessionKey,
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
			req.ProjectPath,
		)
		pluginDir, pluginErr := WriteOpenCodePluginDir(pluginContent, identifier)
		if pluginErr != nil {
			s.cleanupOnFailure(ctx, req.AgentType, req.TicketID, tempFiles, worktreePath, featureBranch, req.ProjectPath)
			return nil, pluginErr
		}
		launcherParams.CleanupDirs = []string{pluginDir}
		launcherParams.EnvVars["OPENCODE_CONFIG_DIR"] = pluginDir
	}

	launcherPath, err := WriteLauncherScript(launcherParams, identifier, s.deps.MCPConfigDir)
	if err != nil {
		s.cleanupOnFailure(ctx, req.AgentType, req.TicketID, tempFiles, worktreePath, featureBranch, req.ProjectPath)
		return nil, err
	}
	allTempFiles := append(tempFiles, launcherPath)

	// Spawn in tmux
	launchCmd := "bash " + launcherPath
	windowIndex, err := s.spawnInTmux(req, windowName, launchCmd, workingDir)
	if err != nil {
		s.cleanupOnFailure(ctx, req.AgentType, req.TicketID, allTempFiles, worktreePath, featureBranch, req.ProjectPath)
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

	// Find cortexd path
	cortexdPath, err := s.getCortexdPath()
	if err != nil {
		return nil, err
	}

	// Determine identifier for file naming
	identifier := req.TicketID
	if identifier == "" {
		if req.AgentType == AgentTypeMeta {
			identifier = "meta"
		} else {
			identifier = "architect-" + req.TmuxSession
		}
	}

	// Generate MCP config
	mcpConfig := GenerateMCPConfig(MCPConfigParams{
		CortexdPath: cortexdPath,
		TicketID:    req.TicketID,
		TicketType:  req.TicketType,
		TicketsDir:  req.TicketsDir,
		ProjectPath: req.ProjectPath,
		TmuxSession: req.TmuxSession,
		IsMeta:      req.AgentType == AgentTypeMeta,
	})

	mcpConfigPath, err := WriteMCPConfig(mcpConfig, identifier, s.deps.MCPConfigDir)
	if err != nil {
		return nil, err
	}

	// Determine env vars based on agent type
	envVars := map[string]string{}
	if req.ProjectPath != "" {
		envVars["CORTEX_PROJECT"] = req.ProjectPath
	}
	switch req.AgentType {
	case AgentTypeTicketAgent:
		envVars["CORTEX_TICKET_ID"] = req.TicketID
	case AgentTypeArchitect:
		envVars["CORTEX_TICKET_ID"] = session.ArchitectSessionKey
	case AgentTypeMeta:
		envVars["CORTEX_TICKET_ID"] = session.MetaSessionKey
	}

	// Generate and write settings config (hooks) - skip for OpenCode (it doesn't support --settings)
	var settingsPath string
	if req.Agent != "opencode" {
		settingsConfig := GenerateSettingsConfig(SettingsConfigParams{
			CortexdPath: cortexdPath,
			TicketID:    identifier,
			ProjectPath: req.ProjectPath,
		})

		settingsPath, err = WriteSettingsConfig(settingsConfig, identifier, s.deps.SettingsConfigDir)
		if err != nil {
			if rmErr := RemoveMCPConfig(mcpConfigPath); rmErr != nil {
				s.logWarn("cleanup: failed to remove MCP config", "path", mcpConfigPath, "error", rmErr)
			}
			return nil, err
		}
	}

	// Generate OpenCode config content for resume (empty system prompt — resume has no prompts)
	var openCodeConfigJSON string
	if req.Agent == "opencode" {
		openCodeConfigJSON, err = GenerateOpenCodeConfigContent(mcpConfig, "")
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

	if req.AgentType == AgentTypeArchitect || req.AgentType == AgentTypeMeta {
		launcherParams.ReplaceSystemPrompt = true
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
			req.ProjectPath,
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

	// Create session in store for architect/meta resume
	if s.deps.SessionStore != nil {
		switch req.AgentType {
		case AgentTypeArchitect:
			if _, createErr := s.deps.SessionStore.CreateArchitect(req.Agent, req.WindowName); createErr != nil {
				s.logWarn("resume: failed to create architect session", "error", createErr)
			}
		case AgentTypeMeta:
			if _, createErr := s.deps.SessionStore.CreateMeta(req.Agent, req.WindowName); createErr != nil {
				s.logWarn("resume: failed to create meta session", "error", createErr)
			}
		}
	}

	// Spawn in tmux
	launchCmd := "bash " + launcherPath
	var windowIndex int

	homeDir, _ := os.UserHomeDir()
	if homeDir == "" {
		homeDir = os.TempDir()
	}

	switch req.AgentType {
	case AgentTypeArchitect:
		err = s.deps.TmuxManager.SpawnArchitect(req.TmuxSession, req.WindowName, launchCmd, "cortex kanban", req.ProjectPath, req.ProjectPath)
	case AgentTypeTicketAgent:
		companionCmd := fmt.Sprintf("CORTEX_TICKET_ID=%s cortex show", req.TicketID)
		windowIndex, err = s.deps.TmuxManager.SpawnAgent(req.TmuxSession, req.WindowName, launchCmd, companionCmd, req.ProjectPath, req.ProjectPath)
	case AgentTypeMeta:
		workDir := req.ProjectPath
		if workDir == "" {
			workDir = homeDir
		}
		err = s.deps.TmuxManager.SpawnArchitect(req.TmuxSession, req.WindowName, launchCmd, "cortex dashboard", workDir, workDir)
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

				// Clean up existing worktree and branch so Spawn() can create fresh ones
				if existingSess.WorktreePath != nil && existingSess.FeatureBranch != nil {
					wm := worktree.NewManager(req.ProjectPath)
					if err := wm.Remove(ctx, *existingSess.WorktreePath, *existingSess.FeatureBranch); err != nil {
						s.logWarn("fresh: failed to clean up old worktree/branch", "error", err)
					}
				}
			}
		case AgentTypeArchitect:
			if err := s.deps.SessionStore.EndArchitect(); err != nil {
				s.logWarn("fresh: failed to end existing architect session", "error", err)
			}
		case AgentTypeMeta:
			if err := s.deps.SessionStore.EndMeta(); err != nil {
				s.logWarn("fresh: failed to end existing meta session", "error", err)
			}
		}
	}

	return s.Spawn(ctx, req)
}

// validateSpawnRequest validates a spawn request.
func (s *Spawner) validateSpawnRequest(req SpawnRequest) error {
	if req.TmuxSession == "" {
		return &ConfigError{Field: "TmuxSession", Message: "cannot be empty"}
	}

	if err := validateTmuxName(req.TmuxSession); err != nil {
		return &ConfigError{Field: "TmuxSession", Message: err.Error()}
	}

	if req.ProjectPath != "" {
		if _, err := os.Stat(req.ProjectPath); os.IsNotExist(err) {
			return &ConfigError{Field: "ProjectPath", Message: "directory does not exist"}
		}
	}

	// Meta agent doesn't require ProjectPath or ProjectName
	if req.AgentType == AgentTypeMeta {
		return nil
	}

	if req.AgentType == AgentTypeTicketAgent {
		if req.TicketID == "" {
			return &ConfigError{Field: "TicketID", Message: "cannot be empty for ticket agent"}
		}
		if req.Ticket == nil {
			return &ConfigError{Field: "Ticket", Message: "cannot be nil for ticket agent"}
		}
	}

	if req.AgentType == AgentTypeArchitect {
		if req.ProjectName == "" {
			return &ConfigError{Field: "ProjectName", Message: "cannot be empty for architect"}
		}
	}

	return nil
}

// tmuxNameRegex matches valid tmux session/window names:
// - alphanumeric characters, underscores, and hyphens only
var tmuxNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// validateTmuxName validates a tmux session or window name.
func validateTmuxName(name string) error {
	if len(name) > 128 {
		return fmt.Errorf("exceeds maximum length of 128 characters")
	}
	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("cannot start with a hyphen")
	}
	if strings.ContainsAny(name, ":.") {
		return fmt.Errorf("cannot contain colons or periods (tmux delimiters)")
	}
	if !tmuxNameRegex.MatchString(name) {
		return fmt.Errorf("must contain only alphanumeric characters, underscores, and hyphens")
	}
	return nil
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

// generateWindowName generates the tmux window name for the agent.
func (s *Spawner) generateWindowName(req SpawnRequest) string {
	if req.AgentType == AgentTypeTicketAgent && req.Ticket != nil {
		return GenerateWindowName(req.Ticket.Title)
	}
	if req.AgentType == AgentTypeMeta {
		return "meta"
	}
	return "architect"
}

// promptInfo contains both dynamic prompt text and the static system prompt content.
type promptInfo struct {
	PromptText          string
	SystemPromptContent string
}

// buildPrompt builds the dynamic prompt and returns the system prompt content.
// Dynamic content (ticket details, ticket lists) is embedded in the prompt.
// Static instructions are loaded from file via --system-prompt (architect, full replace)
// or --append-system-prompt (ticket agent, appended to default).
func (s *Spawner) buildPrompt(req SpawnRequest, worktreePath, featureBranch *string) (*promptInfo, error) {
	switch req.AgentType {
	case AgentTypeTicketAgent:
		return s.buildTicketAgentPrompt(req, worktreePath, featureBranch)
	case AgentTypeArchitect:
		return s.buildArchitectPrompt(req)
	case AgentTypeMeta:
		return s.buildMetaPrompt(req)
	default:
		return nil, &ConfigError{Field: "AgentType", Message: "unknown agent type: " + string(req.AgentType)}
	}
}

// buildTicketAgentPrompt creates the dynamic ticket prompt.
func (s *Spawner) buildTicketAgentPrompt(req SpawnRequest, worktreePath, featureBranch *string) (*promptInfo, error) {
	// Determine ticket type
	ticketType := req.Ticket.Type
	if ticketType == "" {
		ticketType = ticket.DefaultTicketType
	}

	// Create prompt resolver with fallback support
	resolver := prompt.NewPromptResolver(req.ProjectPath, req.BaseConfigPath)

	// Load system prompt (MCP tool instructions and workflow)
	var systemPromptContent string
	{
		var err error
		systemPromptContent, err = resolver.ResolveTicketPrompt(ticketType, prompt.StageSystem)
		if err != nil {
			return nil, err
		}
	}

	// Load kickoff template
	kickoffTemplate, err := resolver.ResolveTicketPrompt(ticketType, prompt.StageKickoff)
	if err != nil {
		// Fall back to simple format if template doesn't exist
		promptText := fmt.Sprintf("# Ticket: %s\n\n%s", req.Ticket.Title, req.Ticket.Body)
		return &promptInfo{
			PromptText:          promptText,
			SystemPromptContent: systemPromptContent,
		}, nil
	}

	// Render template with ticket variables
	vars := prompt.TicketVars{
		ProjectPath: req.ProjectPath,
		TicketID:    req.TicketID,
		TicketTitle: req.Ticket.Title,
		TicketBody:  req.Ticket.Body,
		Comments:    formatTicketComments(req.Ticket.Comments),
		References:  formatTicketReferences(req.Ticket.References),
		IsWorktree:  worktreePath != nil,
	}
	if worktreePath != nil {
		vars.WorktreePath = *worktreePath
	}
	if featureBranch != nil {
		vars.WorktreeBranch = *featureBranch
	}

	promptText, err := prompt.RenderTemplate(kickoffTemplate, vars)
	if err != nil {
		return nil, err
	}

	return &promptInfo{
		PromptText:          promptText,
		SystemPromptContent: systemPromptContent,
	}, nil
}

// formatTicketComments formats ticket comments into a markdown string for prompt injection.
func formatTicketComments(comments []ticket.Comment) string {
	if len(comments) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, c := range comments {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("### [%s] — %s\n", c.Type, c.Created.UTC().Format("2006-01-02 15:04 UTC")))
		sb.WriteString(c.Content)
		sb.WriteString("\n")
	}
	return sb.String()
}

// formatTicketReferences formats ticket references into a bulleted markdown list.
func formatTicketReferences(refs []string) string {
	if len(refs) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, ref := range refs {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("- ")
		sb.WriteString(ref)
	}
	return sb.String()
}

// buildArchitectPrompt creates the dynamic architect prompt with ticket list.
func (s *Spawner) buildArchitectPrompt(req SpawnRequest) (*promptInfo, error) {
	// Create prompt resolver with fallback support
	resolver := prompt.NewPromptResolver(req.ProjectPath, req.BaseConfigPath)

	// Load system prompt (MCP tool instructions and workflow)
	var systemPromptContent string
	{
		var err error
		systemPromptContent, err = resolver.ResolveArchitectPrompt(prompt.StageSystem)
		if err != nil {
			return nil, err
		}
	}

	// Query daemon API to get tickets by status
	client := sdk.DefaultClient(req.ProjectPath)
	tickets, err := client.ListAllTickets("", nil, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list tickets: %w", err)
	}

	// Build formatted ticket list
	var sb strings.Builder

	writeSection := func(name string, items []sdk.TicketSummary) {
		sb.WriteString(fmt.Sprintf("## %s\n", name))
		if len(items) == 0 {
			sb.WriteString("(none)\n")
		} else {
			for _, t := range items {
				dueStr := ""
				if t.Due != nil {
					dueStr = fmt.Sprintf(" (due: %s)", t.Due.Format(time.DateOnly))
				}
				sb.WriteString(fmt.Sprintf("- [%s] %s%s (updated: %s)\n", t.ID, t.Title, dueStr, t.Updated.Format(time.DateOnly)))
			}
		}
		sb.WriteString("\n")
	}

	writeSection("Backlog", tickets.Backlog)
	writeSection("In Progress", tickets.Progress)
	writeSection("Review", tickets.Review)
	doneTickets := tickets.Done
	if len(doneTickets) > 10 {
		doneTickets = doneTickets[:10]
	}
	writeSection("Done", doneTickets)

	ticketList := sb.String()

	// Fetch top tags (graceful degradation)
	var topTags string
	tagsResp, tagsErr := client.ListTags()
	if tagsErr == nil && len(tagsResp.Tags) > 0 {
		limit := min(20, len(tagsResp.Tags))
		tagNames := make([]string, limit)
		for i := range limit {
			tagNames[i] = tagsResp.Tags[i].Name
		}
		topTags = strings.Join(tagNames, ", ")
	}

	// Fetch recent docs (graceful degradation)
	var docsList string
	docsResp, docsErr := client.ListDocs("", "", "")
	if docsErr == nil && len(docsResp.Docs) > 0 {
		// Sort by Created descending (RFC3339 strings sort correctly)
		sorted := make([]sdk.DocSummary, len(docsResp.Docs))
		copy(sorted, docsResp.Docs)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Created > sorted[j].Created
		})
		limit := min(20, len(sorted))
		var docSB strings.Builder
		for i := range limit {
			d := sorted[i]
			docSB.WriteString(fmt.Sprintf("- [%s] %s (%s, created: %s)\n", d.ID, d.Title, d.Category, d.Created))
		}
		docsList = docSB.String()
	}

	// Try to load and render KICKOFF template
	kickoffTemplate, kickoffErr := resolver.ResolveArchitectPrompt(prompt.StageKickoff)
	if kickoffErr == nil {
		vars := prompt.ArchitectKickoffVars{
			ProjectName: req.ProjectName,
			TicketList:  ticketList,
			CurrentDate: time.Now().Format("2006-01-02 15:04 MST"),
			TopTags:     topTags,
			DocsList:    docsList,
		}
		rendered, renderErr := prompt.RenderTemplate(kickoffTemplate, vars)
		if renderErr == nil {
			return &promptInfo{
				PromptText:          rendered,
				SystemPromptContent: systemPromptContent,
			}, nil
		}
	}

	// Fallback: inline format
	promptText := fmt.Sprintf("# Project: %s\n\n# Tickets\n\n%s", req.ProjectName, ticketList)

	return &promptInfo{
		PromptText:          promptText,
		SystemPromptContent: systemPromptContent,
	}, nil
}

// buildMetaPrompt creates the dynamic meta prompt with project and session listing.
func (s *Spawner) buildMetaPrompt(req SpawnRequest) (*promptInfo, error) {
	// Create prompt resolver — meta prompts live in defaults (base config path)
	resolver := prompt.NewPromptResolver("", req.BaseConfigPath)

	// Load system prompt
	var systemPromptContent string
	{
		var err error
		systemPromptContent, err = resolver.ResolveMetaPrompt(prompt.StageSystem)
		if err != nil {
			return nil, err
		}
	}

	// Query daemon API to get all projects
	client := sdk.DefaultClient("")
	projectsResp, err := client.ListProjects()
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	// Build project list
	var projSB strings.Builder
	for _, p := range projectsResp.Projects {
		status := "exists"
		if !p.Exists {
			status = "missing"
		}
		projSB.WriteString(fmt.Sprintf("## %s (%s)\n", p.Title, p.Path))
		projSB.WriteString(fmt.Sprintf("  - status: %s\n", status))
		if p.Counts != nil {
			projSB.WriteString(fmt.Sprintf("  - backlog: %d, progress: %d, review: %d, done: %d\n",
				p.Counts.Backlog, p.Counts.Progress, p.Counts.Review, p.Counts.Done))
		}
		projSB.WriteString("\n")
	}

	// Build session list across all projects (best-effort)
	var sessSB strings.Builder
	for _, p := range projectsResp.Projects {
		if !p.Exists {
			continue
		}
		sessResp, sessErr := client.WithProject(p.Path).ListSessions()
		if sessErr != nil || len(sessResp.Sessions) == 0 {
			continue
		}
		sessSB.WriteString(fmt.Sprintf("### %s\n", p.Title))
		for _, sess := range sessResp.Sessions {
			sessSB.WriteString(fmt.Sprintf("- [%s] %s (%s) — %s\n", sess.SessionType, sess.TicketTitle, sess.Agent, sess.Status))
		}
		sessSB.WriteString("\n")
	}

	// Try to load and render KICKOFF template
	kickoffTemplate, kickoffErr := resolver.ResolveMetaPrompt(prompt.StageKickoff)
	if kickoffErr == nil {
		vars := prompt.MetaKickoffVars{
			CurrentDate: time.Now().Format("2006-01-02 15:04 MST"),
			ProjectList: projSB.String(),
			SessionList: sessSB.String(),
		}
		rendered, renderErr := prompt.RenderTemplate(kickoffTemplate, vars)
		if renderErr == nil {
			return &promptInfo{
				PromptText:          rendered,
				SystemPromptContent: systemPromptContent,
			}, nil
		}
	}

	// Fallback: inline format
	promptText := fmt.Sprintf("# Cortex Meta Session\n\n# Projects\n\n%s", projSB.String())

	return &promptInfo{
		PromptText:          promptText,
		SystemPromptContent: systemPromptContent,
	}, nil
}

// spawnInTmux spawns the agent in a tmux window.
func (s *Spawner) spawnInTmux(req SpawnRequest, windowName, launchCmd, workingDir string) (int, error) {
	switch req.AgentType {
	case AgentTypeTicketAgent:
		// Companion command shows ticket details
		companionCmd := fmt.Sprintf("CORTEX_TICKET_ID=%s cortex show", req.TicketID)
		return s.deps.TmuxManager.SpawnAgent(req.TmuxSession, windowName, launchCmd, companionCmd, workingDir, req.ProjectPath)
	case AgentTypeArchitect:
		// Companion command shows kanban board
		err := s.deps.TmuxManager.SpawnArchitect(req.TmuxSession, windowName, launchCmd, "cortex kanban", workingDir, req.ProjectPath)
		return 0, err
	case AgentTypeMeta:
		// Companion command shows project dashboard
		err := s.deps.TmuxManager.SpawnArchitect(req.TmuxSession, windowName, launchCmd, "cortex dashboard", workingDir, workingDir)
		return 0, err
	default:
		return 0, &ConfigError{Field: "AgentType", Message: "unknown agent type: " + string(req.AgentType)}
	}
}

// cleanupOnFailure cleans up resources when spawn fails.
func (s *Spawner) cleanupOnFailure(ctx context.Context, agentType AgentType, ticketID string, tempFiles []string, worktreePath, featureBranch *string, projectPath string) {
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
		case AgentTypeMeta:
			if err := s.deps.SessionStore.EndMeta(); err != nil {
				s.logWarn("cleanup: failed to end meta session", "error", err)
			}
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
	if worktreePath != nil && featureBranch != nil && projectPath != "" {
		wm := worktree.NewManager(projectPath)
		if err := wm.Remove(ctx, *worktreePath, *featureBranch); err != nil {
			s.logWarn("cleanup: failed to remove worktree", "path", *worktreePath, "error", err)
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
