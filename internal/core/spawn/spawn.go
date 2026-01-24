package spawn

import (
	"fmt"
	"strings"
	"time"

	"github.com/kareemaly/cortex/internal/binpath"
	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/prompt"
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
	SetSession(ticketID, agent, tmuxWindow string) (*ticket.Session, error)
	EndSession(ticketID string) error
}

// TmuxManagerInterface defines the tmux operations needed for spawning.
type TmuxManagerInterface interface {
	WindowExists(session, windowName string) (bool, error)
	SpawnAgent(session, windowName, agentCommand, companionCommand, workingDir string) (int, error)
	SpawnArchitect(session, windowName, agentCommand, companionCommand, workingDir string) error
}

// Dependencies contains the external dependencies for the Spawner.
type Dependencies struct {
	Store             StoreInterface
	TmuxManager       TmuxManagerInterface
	CortexdPath       string // optional override for cortexd binary path
	MCPConfigDir      string // optional override for MCP config directory
	SettingsConfigDir string // optional override for settings config directory
}

// Spawner handles spawning agent sessions.
type Spawner struct {
	deps Dependencies
}

// NewSpawner creates a new Spawner with the given dependencies.
func NewSpawner(deps Dependencies) *Spawner {
	return &Spawner{deps: deps}
}

// SpawnRequest contains parameters for spawning a new agent session.
type SpawnRequest struct {
	AgentType   AgentType
	Agent       string // agent identifier (e.g., "claude")
	TmuxSession string
	ProjectPath string
	TicketsDir  string

	// For ticket agents
	TicketID string
	Ticket   *ticket.Ticket

	// For architect agents
	ProjectName string
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
func (s *Spawner) Spawn(req SpawnRequest) (*SpawnResult, error) {
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

	// Create session in store (ticket agents only)
	var sessionID string
	if req.AgentType == AgentTypeTicketAgent {
		session, err := s.deps.Store.SetSession(req.TicketID, req.Agent, windowName)
		if err != nil {
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
		s.cleanupOnFailure(req.AgentType, req.TicketID, "", "")
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
		s.cleanupOnFailure(req.AgentType, req.TicketID, mcpConfigPath, "")
		return nil, err
	}

	// Load and build prompt
	pInfo, err := s.buildPrompt(req)
	if err != nil {
		s.cleanupOnFailure(req.AgentType, req.TicketID, mcpConfigPath, settingsPath)
		return &SpawnResult{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	// Build claude command with different options based on agent type
	var claudeCmd string
	switch req.AgentType {
	case AgentTypeArchitect:
		// Architects use allowed tools - listTickets and readTicket are auto-approved
		// Other tools (createTicket, updateTicket, deleteTicket, moveTicket, spawnSession) require user approval
		claudeCmd = BuildClaudeCommand(ClaudeCommandParams{
			Prompt:             pInfo.PromptText,
			AppendSystemPrompt: pInfo.SystemPromptPath,
			MCPConfigPath:      mcpConfigPath,
			SettingsPath:       settingsPath,
			AllowedTools:       []string{"mcp__cortex__listTickets", "mcp__cortex__readTicket"},
			SessionID:          sessionID,
		})
	case AgentTypeTicketAgent:
		// Ticket agents use plan mode
		claudeCmd = BuildClaudeCommand(ClaudeCommandParams{
			Prompt:             pInfo.PromptText,
			AppendSystemPrompt: pInfo.SystemPromptPath,
			MCPConfigPath:      mcpConfigPath,
			SettingsPath:       settingsPath,
			PermissionMode:     "plan",
			SessionID:          sessionID,
		})
	}

	// Spawn in tmux
	windowIndex, err := s.spawnInTmux(req, windowName, claudeCmd)
	if err != nil {
		s.cleanupOnFailure(req.AgentType, req.TicketID, mcpConfigPath, settingsPath)
		return &SpawnResult{
			Success: false,
			Message: "failed to spawn agent in tmux: " + err.Error(),
		}, nil
	}

	return &SpawnResult{
		Success:       true,
		SessionID:     sessionID,
		TmuxWindow:    windowName,
		WindowIndex:   windowIndex,
		MCPConfigPath: mcpConfigPath,
		SettingsPath:  settingsPath,
		Message:       "Agent session spawned in tmux window '" + windowName + "'",
	}, nil
}

// Resume resumes an orphaned ticket agent session.
// Note: This is only used for ticket agents - architect sessions do not support resume.
func (s *Spawner) Resume(req ResumeRequest) (*SpawnResult, error) {
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
		_ = RemoveMCPConfig(mcpConfigPath)
		return nil, err
	}

	// Build resume command (no prompt needed, just resume ID)
	claudeCmd := BuildClaudeCommand(ClaudeCommandParams{
		Prompt:         "",
		MCPConfigPath:  mcpConfigPath,
		SettingsPath:   settingsPath,
		PermissionMode: "plan",
		ResumeID:       req.SessionID,
	})

	// Set up command with ticket environment variables
	cmdWithEnv := fmt.Sprintf("CORTEX_TICKET_ID=%s CORTEX_PROJECT=%s %s", req.TicketID, req.ProjectPath, claudeCmd)
	companionCmd := fmt.Sprintf("CORTEX_TICKET_ID=%s cortex show", req.TicketID)

	// Spawn in tmux
	windowIndex, err := s.deps.TmuxManager.SpawnAgent(req.TmuxSession, req.WindowName, cmdWithEnv, companionCmd, req.ProjectPath)
	if err != nil {
		_ = RemoveMCPConfig(mcpConfigPath)
		_ = RemoveSettingsConfig(settingsPath)
		return &SpawnResult{
			Success: false,
			Message: "failed to spawn agent in tmux: " + err.Error(),
		}, nil
	}

	return &SpawnResult{
		Success:       true,
		TmuxWindow:    req.WindowName,
		WindowIndex:   windowIndex,
		MCPConfigPath: mcpConfigPath,
		SettingsPath:  settingsPath,
		Message:       "Session resumed in tmux window '" + req.WindowName + "'",
	}, nil
}

// Fresh clears any existing session and spawns a new one.
func (s *Spawner) Fresh(req SpawnRequest) (*SpawnResult, error) {
	// End existing session if present
	if req.AgentType == AgentTypeTicketAgent && req.Ticket != nil && req.Ticket.Session != nil {
		_ = s.deps.Store.EndSession(req.TicketID)
	}

	return s.Spawn(req)
}

// validateSpawnRequest validates a spawn request.
func (s *Spawner) validateSpawnRequest(req SpawnRequest) error {
	if req.TmuxSession == "" {
		return &ConfigError{Field: "TmuxSession", Message: "cannot be empty"}
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

// promptInfo contains both dynamic prompt text and the static system prompt path.
type promptInfo struct {
	PromptText       string
	SystemPromptPath string
}

// buildPrompt builds the dynamic prompt and returns the system prompt path.
// Dynamic content (ticket details, ticket lists) is embedded in the prompt.
// Static instructions are loaded from file via --append-system-prompt.
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
	systemPromptPath := prompt.TicketAgentPath(req.ProjectPath)
	if err := prompt.ValidatePromptFile(systemPromptPath); err != nil {
		return nil, err
	}

	promptText := fmt.Sprintf("# Ticket: %s\n\n%s", req.Ticket.Title, req.Ticket.Body)

	return &promptInfo{
		PromptText:       promptText,
		SystemPromptPath: systemPromptPath,
	}, nil
}

// buildArchitectPrompt creates the dynamic architect prompt with ticket list.
func (s *Spawner) buildArchitectPrompt(req SpawnRequest) (*promptInfo, error) {
	systemPromptPath := prompt.ArchitectPath(req.ProjectPath)
	if err := prompt.ValidatePromptFile(systemPromptPath); err != nil {
		return nil, err
	}

	// Query daemon API to get tickets by status
	client := sdk.DefaultClient(req.ProjectPath)
	tickets, err := client.ListAllTickets("")
	if err != nil {
		return nil, fmt.Errorf("failed to list tickets: %w", err)
	}

	// Build formatted ticket list with project name
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Project: %s\n\n", req.ProjectName))
	sb.WriteString("# Tickets\n\n")

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
	writeSection("Done", tickets.Done)

	return &promptInfo{
		PromptText:       sb.String(),
		SystemPromptPath: systemPromptPath,
	}, nil
}

// spawnInTmux spawns the agent in a tmux window.
func (s *Spawner) spawnInTmux(req SpawnRequest, windowName, claudeCmd string) (int, error) {
	switch req.AgentType {
	case AgentTypeTicketAgent:
		// Prefix command with env vars so child processes can identify the ticket and project
		cmdWithEnv := fmt.Sprintf("CORTEX_TICKET_ID=%s CORTEX_PROJECT=%s %s", req.TicketID, req.ProjectPath, claudeCmd)
		// Companion command shows ticket details
		companionCmd := fmt.Sprintf("CORTEX_TICKET_ID=%s cortex show", req.TicketID)
		return s.deps.TmuxManager.SpawnAgent(req.TmuxSession, windowName, cmdWithEnv, companionCmd, req.ProjectPath)
	case AgentTypeArchitect:
		// Companion command shows kanban board
		err := s.deps.TmuxManager.SpawnArchitect(req.TmuxSession, windowName, claudeCmd, "cortex kanban", req.ProjectPath)
		return 0, err
	default:
		return 0, &ConfigError{Field: "AgentType", Message: "unknown agent type: " + string(req.AgentType)}
	}
}

// cleanupOnFailure cleans up resources when spawn fails.
func (s *Spawner) cleanupOnFailure(agentType AgentType, ticketID, mcpConfigPath, settingsPath string) {
	if agentType == AgentTypeTicketAgent && ticketID != "" {
		_ = s.deps.Store.EndSession(ticketID)
	}
	if mcpConfigPath != "" {
		_ = RemoveMCPConfig(mcpConfigPath)
	}
	if settingsPath != "" {
		_ = RemoveSettingsConfig(settingsPath)
	}
}
