package spawn

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	architectconfig "github.com/kareemaly/cortex/internal/architect/config"
	"github.com/kareemaly/cortex/internal/binpath"
	"github.com/kareemaly/cortex/internal/cli/sdk"
	daemonconfig "github.com/kareemaly/cortex/internal/daemon/config"
	"github.com/kareemaly/cortex/internal/prompt"
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
	})

	identifier := req.TicketID
	if identifier == "" {
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
			"CORTEX_TICKET_ID": session.ArchitectSessionKey,
			"CORTEX_ARCHITECT": req.ArchitectPath,
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
		if companionCmd == "" {
			companionCmd = fmt.Sprintf("cortex ticket show %s", req.TicketID)
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

// validateSpawnRequest validates a spawn request.
func (s *Spawner) validateSpawnRequest(req SpawnRequest) error {
	if req.TmuxSession == "" {
		return &ConfigError{Field: "TmuxSession", Message: "cannot be empty"}
	}

	if err := validateTmuxName(req.TmuxSession); err != nil {
		return &ConfigError{Field: "TmuxSession", Message: err.Error()}
	}

	if req.ArchitectPath != "" {
		if _, err := os.Stat(req.ArchitectPath); os.IsNotExist(err) {
			return &ConfigError{Field: "ProjectPath", Message: "directory does not exist"}
		}
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
		if req.ArchitectName == "" {
			return &ConfigError{Field: "ArchitectName", Message: "cannot be empty for architect"}
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

// getWorkingDirectory determines the working directory for a ticket agent.
// Work tickets spawn in the repo directory, research tickets spawn in the architect project root.
func getWorkingDirectory(req SpawnRequest) (string, error) {
	if req.AgentType != AgentTypeTicketAgent {
		return req.ArchitectPath, nil
	}

	if req.Ticket == nil {
		return req.ArchitectPath, nil
	}

	if req.Ticket.Type == "work" {
		if req.Ticket.Repo != "" {
			repo := req.Ticket.Repo
			if strings.HasPrefix(repo, "~/") {
				if home, err := os.UserHomeDir(); err == nil {
					repo = filepath.Join(home, repo[2:])
				}
			}
			// Validate repo exists and is a git repository
			if err := validateGitRepository(repo); err != nil {
				return "", err
			}
			return repo, nil
		}
		return req.ArchitectPath, nil
	}

	if req.Ticket.Type == "research" {
		if req.Ticket.Path != "" {
			path := req.Ticket.Path
			if strings.HasPrefix(path, "~/") {
				if home, err := os.UserHomeDir(); err == nil {
					path = filepath.Join(home, path[2:])
				}
			}
			// Validate directory exists (no git check required for research paths)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return "", &ConfigError{
					Field:   "Path",
					Message: fmt.Sprintf("research path directory does not exist: %s", path),
				}
			}
			return path, nil
		}
		return req.ArchitectPath, nil
	}

	return req.ArchitectPath, nil
}

// validateGitRepository checks if the given path exists and is a git repository.
func validateGitRepository(repoPath string) error {
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return &ConfigError{
			Field:   "Repo",
			Message: fmt.Sprintf("repository directory does not exist: %s", repoPath),
		}
	}

	// Check for .git directory
	gitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return &ConfigError{
			Field:   "Repo",
			Message: fmt.Sprintf("not a git repository: %s", repoPath),
		}
	}

	return nil
}

// generateWindowName generates the tmux window name for the agent.
func (s *Spawner) generateWindowName(req SpawnRequest) string {
	if req.AgentType == AgentTypeTicketAgent && req.Ticket != nil {
		return GenerateWindowName(req.Ticket.Title)
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
func (s *Spawner) buildPrompt(req SpawnRequest) (*promptInfo, error) {
	switch req.AgentType {
	case AgentTypeTicketAgent:
		return s.buildTicketAgentPrompt(req)
	case AgentTypeArchitect:
		return s.buildArchitectPrompt(req)
	default:
		return nil, &ConfigError{Field: "AgentType", Message: "unknown agent type: " + string(req.AgentType)}
	}
}

// buildTicketAgentPrompt creates the dynamic ticket prompt.
func (s *Spawner) buildTicketAgentPrompt(req SpawnRequest) (*promptInfo, error) {
	// Determine ticket type
	ticketType := req.Ticket.Type
	if ticketType == "" {
		ticketType = ticket.DefaultTicketType
	}

	// Create prompt resolver with fallback to defaults
	resolver := prompt.NewPromptResolver(req.ArchitectPath, s.deps.DefaultsDir)

	// Try to resolve system prompt (optional — used by opencode via instructions file)
	systemPromptContent, _ := resolver.ResolveTicketPrompt(ticketType, prompt.StageSystem)

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
		ProjectPath: req.ArchitectPath,
		TicketID:    req.TicketID,
		TicketTitle: req.Ticket.Title,
		TicketBody:  req.Ticket.Body,
		References:  formatTicketReferences(req.Ticket.References),
		Repo:        req.Ticket.Repo,
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
	// Create prompt resolver with fallback to defaults
	resolver := prompt.NewPromptResolver(req.ArchitectPath, s.deps.DefaultsDir)

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
	client := sdk.DefaultClient(req.ArchitectPath)
	tickets, err := client.ListAllTickets("", nil)
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
	doneTickets := tickets.Done
	if len(doneTickets) > 10 {
		doneTickets = doneTickets[:10]
	}
	writeSection("Done", doneTickets)

	ticketList := sb.String()

	// Fetch conclusions (graceful degradation)
	var sessionsList string
	conclusionsResp, conclusionsErr := client.ListConclusions(sdk.ListConclusionsParams{Limit: 10})
	if conclusionsErr == nil && len(conclusionsResp.Conclusions) > 0 {
		var sessionsSB strings.Builder
		for _, c := range conclusionsResp.Conclusions {
			sessionsSB.WriteString(fmt.Sprintf("- [%s] %s (%s)\n", c.ID, c.Ticket, c.Type))
		}
		sessionsList = sessionsSB.String()
	}

	// Fetch repos from project config (graceful degradation)
	var reposList string
	projectCfg, cfgErr := architectconfig.Load(req.ArchitectPath)
	if cfgErr == nil && len(projectCfg.Repos) > 0 {
		var reposSB strings.Builder
		for _, repo := range projectCfg.Repos {
			reposSB.WriteString(fmt.Sprintf("- %s\n", repo))
		}
		reposList = reposSB.String()
	}

	// Try to load and render KICKOFF template
	kickoffTemplate, kickoffErr := resolver.ResolveArchitectPrompt(prompt.StageKickoff)
	if kickoffErr == nil {
		vars := prompt.ArchitectKickoffVars{
			ArchitectName: req.ArchitectName,
			TicketList:    ticketList,
			CurrentDate:   time.Now().Format("2006-01-02 15:04 MST"),
			Sessions:      sessionsList,
			Repos:         reposList,
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
	promptText := fmt.Sprintf("# Project: %s\n\n# Tickets\n\n%s", req.ArchitectName, ticketList)

	return &promptInfo{
		PromptText:          promptText,
		SystemPromptContent: systemPromptContent,
	}, nil
}

// spawnInTmux spawns the agent in a tmux window.
func (s *Spawner) spawnInTmux(req SpawnRequest, windowName, launchCmd, workingDir string) (int, error) {
	switch req.AgentType {
	case AgentTypeTicketAgent:
		// Use companion command from config, or default to showing ticket details
		companionCmd := req.Companion
		if companionCmd == "" {
			companionCmd = fmt.Sprintf("cortex ticket show %s", req.TicketID)
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
