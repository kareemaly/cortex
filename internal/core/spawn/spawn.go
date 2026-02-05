package spawn

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kareemaly/cortex/internal/binpath"
	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/prompt"
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
)

// StoreInterface defines the ticket store operations needed for spawning.
type StoreInterface interface {
	Get(id string) (*ticket.Ticket, ticket.Status, error)
	SetSession(ticketID, agent, tmuxWindow string, worktreePath, featureBranch *string) (*ticket.Session, error)
	EndSession(ticketID string) error
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
	TmuxSession string
	ProjectPath string
	TicketsDir  string
	SessionID   string
	WindowName  string

	// For ticket agents
	TicketID string

	// Extra CLI args appended to the agent command
	AgentArgs []string
}

// SpawnResult contains the result of a spawn operation.
type SpawnResult struct {
	Success       bool
	TicketID      string
	SessionID     string
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
		stateInfo, err := DetectTicketState(req.Ticket, req.TmuxSession, s.deps.TmuxManager)
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
		slug := ticket.GenerateSlug(req.Ticket.Title)

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

	// Create session in store (ticket agents only)
	var sessionID string
	if req.AgentType == AgentTypeTicketAgent {
		session, err := s.deps.Store.SetSession(req.TicketID, req.Agent, windowName, worktreePath, featureBranch)
		if err != nil {
			if worktreePath != nil && featureBranch != nil {
				wm := worktree.NewManager(req.ProjectPath)
				_ = wm.Remove(ctx, *worktreePath, *featureBranch)
			}
			return nil, err
		}
		sessionID = session.ID
	}

	// Generate and write MCP config
	mcpConfig := GenerateMCPConfig(MCPConfigParams{
		CortexdPath: cortexdPath,
		TicketID:    req.TicketID,
		TicketsDir:  req.TicketsDir,
		ProjectPath: req.ProjectPath,
		TmuxSession: req.TmuxSession,
	})

	identifier := req.TicketID
	if identifier == "" {
		identifier = "architect"
	}

	mcpConfigPath, err := WriteMCPConfig(mcpConfig, identifier, s.deps.MCPConfigDir)
	if err != nil {
		s.cleanupOnFailure(ctx, req.AgentType, req.TicketID, nil, worktreePath, featureBranch, req.ProjectPath)
		return nil, err
	}

	// Generate and write settings config (hooks)
	settingsConfig := GenerateSettingsConfig(SettingsConfigParams{
		CortexdPath: cortexdPath,
		TicketID:    req.TicketID,
		ProjectPath: req.ProjectPath,
	})

	settingsPath, err := WriteSettingsConfig(settingsConfig, identifier, s.deps.SettingsConfigDir)
	if err != nil {
		s.cleanupOnFailure(ctx, req.AgentType, req.TicketID, []string{mcpConfigPath}, worktreePath, featureBranch, req.ProjectPath)
		return nil, err
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

	// Build launcher params based on agent type
	tempFiles := nonEmptyStrings(mcpConfigPath, settingsPath, promptFilePath, systemPromptFilePath)
	launcherParams := LauncherParams{
		PromptFilePath:       promptFilePath,
		SystemPromptFilePath: systemPromptFilePath,
		MCPConfigPath:        mcpConfigPath,
		SettingsPath:         settingsPath,
		SessionID:            sessionID,
		AgentArgs:            req.AgentArgs,
		CleanupFiles:         tempFiles,
	}

	switch req.AgentType {
	case AgentTypeArchitect:
		launcherParams.ReplaceSystemPrompt = true
	case AgentTypeTicketAgent:
		launcherParams.EnvVars = map[string]string{
			"CORTEX_TICKET_ID": req.TicketID,
			"CORTEX_PROJECT":   req.ProjectPath,
		}
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
		SessionID:   sessionID,
		TmuxWindow:  windowName,
		WindowIndex: windowIndex,
		Message:     "Agent session spawned in tmux window '" + windowName + "'",
	}, nil
}

// Resume resumes an orphaned ticket agent session.
// Note: This is only used for ticket agents - architect sessions do not support resume.
func (s *Spawner) Resume(ctx context.Context, req ResumeRequest) (*SpawnResult, error) {
	if req.SessionID == "" {
		return nil, &ConfigError{Field: "SessionID", Message: "cannot be empty for resume"}
	}
	if req.TicketID == "" {
		return nil, &ConfigError{Field: "TicketID", Message: "cannot be empty for resume (only ticket agents support resume)"}
	}

	// Find cortexd path
	cortexdPath, err := s.getCortexdPath()
	if err != nil {
		return nil, err
	}

	// Generate MCP config
	mcpConfig := GenerateMCPConfig(MCPConfigParams{
		CortexdPath: cortexdPath,
		TicketID:    req.TicketID,
		TicketsDir:  req.TicketsDir,
		ProjectPath: req.ProjectPath,
		TmuxSession: req.TmuxSession,
	})

	mcpConfigPath, err := WriteMCPConfig(mcpConfig, req.TicketID, s.deps.MCPConfigDir)
	if err != nil {
		return nil, err
	}

	// Generate and write settings config (hooks)
	settingsConfig := GenerateSettingsConfig(SettingsConfigParams{
		CortexdPath: cortexdPath,
		TicketID:    req.TicketID,
		ProjectPath: req.ProjectPath,
	})

	settingsPath, err := WriteSettingsConfig(settingsConfig, req.TicketID, s.deps.SettingsConfigDir)
	if err != nil {
		if rmErr := RemoveMCPConfig(mcpConfigPath); rmErr != nil {
			s.logWarn("cleanup: failed to remove MCP config", "path", mcpConfigPath, "error", rmErr)
		}
		return nil, err
	}

	// Build launcher script for resume (no prompt files needed)
	tempFiles := nonEmptyStrings(mcpConfigPath, settingsPath)
	launcherParams := LauncherParams{
		MCPConfigPath: mcpConfigPath,
		SettingsPath:  settingsPath,
		ResumeID:      req.SessionID,
		AgentArgs:     req.AgentArgs,
		EnvVars: map[string]string{
			"CORTEX_TICKET_ID": req.TicketID,
			"CORTEX_PROJECT":   req.ProjectPath,
		},
		CleanupFiles: tempFiles,
	}

	launcherPath, err := WriteLauncherScript(launcherParams, req.TicketID, s.deps.MCPConfigDir)
	if err != nil {
		for _, path := range tempFiles {
			if rmErr := os.Remove(path); rmErr != nil && !os.IsNotExist(rmErr) {
				s.logWarn("cleanup: failed to remove temp file", "path", path, "error", rmErr)
			}
		}
		return nil, err
	}
	allTempFiles := append(tempFiles, launcherPath)

	// Spawn in tmux
	launchCmd := "bash " + launcherPath
	companionCmd := fmt.Sprintf("CORTEX_TICKET_ID=%s cortex show", req.TicketID)

	windowIndex, err := s.deps.TmuxManager.SpawnAgent(req.TmuxSession, req.WindowName, launchCmd, companionCmd, req.ProjectPath, req.ProjectPath)
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
	// End existing session if present
	if req.AgentType == AgentTypeTicketAgent && req.Ticket != nil && req.Ticket.Session != nil {
		if err := s.deps.Store.EndSession(req.TicketID); err != nil {
			s.logWarn("fresh: failed to end existing session", "ticketID", req.TicketID, "error", err)
		}

		// Clean up existing worktree and branch so Spawn() can create fresh ones
		session := req.Ticket.Session
		if session.WorktreePath != nil && session.FeatureBranch != nil {
			wm := worktree.NewManager(req.ProjectPath)
			if err := wm.Remove(ctx, *session.WorktreePath, *session.FeatureBranch); err != nil {
				s.logWarn("fresh: failed to clean up old worktree/branch", "error", err)
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
	systemPromptContent, err := resolver.ResolveTicketPrompt(ticketType, prompt.StageSystem)
	if err != nil {
		return nil, err
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

// buildArchitectPrompt creates the dynamic architect prompt with ticket list.
func (s *Spawner) buildArchitectPrompt(req SpawnRequest) (*promptInfo, error) {
	// Create prompt resolver with fallback support
	resolver := prompt.NewPromptResolver(req.ProjectPath, req.BaseConfigPath)

	systemPromptContent, err := resolver.ResolveArchitectPrompt(prompt.StageSystem)
	if err != nil {
		return nil, err
	}

	// Query daemon API to get tickets by status
	client := sdk.DefaultClient(req.ProjectPath)
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
				sb.WriteString(fmt.Sprintf("- [%s] %s (updated: %s)\n", t.ID, t.Title, t.Updated.Format(time.DateOnly)))
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

	// Try to load and render KICKOFF template
	kickoffTemplate, kickoffErr := resolver.ResolveArchitectPrompt(prompt.StageKickoff)
	if kickoffErr == nil {
		vars := prompt.ArchitectKickoffVars{
			ProjectName: req.ProjectName,
			TicketList:  ticketList,
			CurrentDate: time.Now().Format("2006-01-02"),
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
	default:
		return 0, &ConfigError{Field: "AgentType", Message: "unknown agent type: " + string(req.AgentType)}
	}
}

// cleanupOnFailure cleans up resources when spawn fails.
func (s *Spawner) cleanupOnFailure(ctx context.Context, agentType AgentType, ticketID string, tempFiles []string, worktreePath, featureBranch *string, projectPath string) {
	if agentType == AgentTypeTicketAgent && ticketID != "" {
		if err := s.deps.Store.EndSession(ticketID); err != nil {
			s.logWarn("cleanup: failed to end session", "ticketID", ticketID, "error", err)
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
