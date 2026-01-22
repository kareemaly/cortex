package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kareemaly/cortex/internal/binpath"
	"github.com/kareemaly/cortex/internal/prompt"
	"github.com/kareemaly/cortex/internal/ticket"
	"github.com/kareemaly/cortex/internal/tmux"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerArchitectTools registers all tools available to architect sessions.
func (s *Server) registerArchitectTools() {
	// List tickets
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "listTickets",
		Description: "List tickets with optional status and query filters",
	}, s.handleListTickets)

	// Read ticket
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "readTicket",
		Description: "Read full ticket details by ID",
	}, s.handleReadTicket)

	// Create ticket
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "createTicket",
		Description: "Create a new ticket in backlog",
	}, s.handleCreateTicket)

	// Update ticket
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "updateTicket",
		Description: "Update ticket title and/or body",
	}, s.handleUpdateTicket)

	// Delete ticket
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "deleteTicket",
		Description: "Delete a ticket by ID",
	}, s.handleDeleteTicket)

	// Move ticket
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "moveTicket",
		Description: "Move a ticket to a different status",
	}, s.handleMoveTicket)

	// Spawn session (stub)
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "spawnSession",
		Description: "Spawn a new agent session for a ticket",
	}, s.handleSpawnSession)
}

// handleListTickets lists tickets with optional status and query filters.
func (s *Server) handleListTickets(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ListTicketsInput,
) (*mcp.CallToolResult, ListTicketsOutput, error) {
	// Initialize as empty slice (not nil) to ensure JSON marshals to [] not null
	summaries := []TicketSummary{}

	// Prepare query for case-insensitive matching
	query := strings.ToLower(input.Query)

	if input.Status != "" {
		// List by specific status
		status := ticket.Status(input.Status)
		tickets, err := s.store.List(status)
		if err != nil {
			return nil, ListTicketsOutput{}, WrapTicketError(err)
		}
		for _, t := range tickets {
			// Apply query filter if specified
			if query != "" &&
				!strings.Contains(strings.ToLower(t.Title), query) &&
				!strings.Contains(strings.ToLower(t.Body), query) {
				continue
			}
			summaries = append(summaries, ToTicketSummary(t, status))
		}
	} else {
		// List all tickets
		allTickets, err := s.store.ListAll()
		if err != nil {
			return nil, ListTicketsOutput{}, WrapTicketError(err)
		}
		for status, tickets := range allTickets {
			for _, t := range tickets {
				// Apply query filter if specified
				if query != "" &&
					!strings.Contains(strings.ToLower(t.Title), query) &&
					!strings.Contains(strings.ToLower(t.Body), query) {
					continue
				}
				summaries = append(summaries, ToTicketSummary(t, status))
			}
		}
	}

	return nil, ListTicketsOutput{
		Tickets: summaries,
		Total:   len(summaries),
	}, nil
}

// handleReadTicket reads a ticket by ID.
func (s *Server) handleReadTicket(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ReadTicketInput,
) (*mcp.CallToolResult, ReadTicketOutput, error) {
	if input.ID == "" {
		return nil, ReadTicketOutput{}, NewValidationError("id", "cannot be empty")
	}

	t, status, err := s.store.Get(input.ID)
	if err != nil {
		return nil, ReadTicketOutput{}, WrapTicketError(err)
	}

	return nil, ReadTicketOutput{
		Ticket: ToTicketOutput(t, status),
	}, nil
}

// handleCreateTicket creates a new ticket.
func (s *Server) handleCreateTicket(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input CreateTicketInput,
) (*mcp.CallToolResult, CreateTicketOutput, error) {
	t, err := s.store.Create(input.Title, input.Body)
	if err != nil {
		return nil, CreateTicketOutput{}, WrapTicketError(err)
	}

	return nil, CreateTicketOutput{
		Ticket: ToTicketOutput(t, ticket.StatusBacklog),
	}, nil
}

// handleUpdateTicket updates a ticket's title and/or body.
func (s *Server) handleUpdateTicket(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input UpdateTicketInput,
) (*mcp.CallToolResult, UpdateTicketOutput, error) {
	if input.ID == "" {
		return nil, UpdateTicketOutput{}, NewValidationError("id", "cannot be empty")
	}

	t, err := s.store.Update(input.ID, input.Title, input.Body)
	if err != nil {
		return nil, UpdateTicketOutput{}, WrapTicketError(err)
	}

	// Get status
	_, status, err := s.store.Get(input.ID)
	if err != nil {
		return nil, UpdateTicketOutput{}, WrapTicketError(err)
	}

	return nil, UpdateTicketOutput{
		Ticket: ToTicketOutput(t, status),
	}, nil
}

// handleDeleteTicket deletes a ticket.
func (s *Server) handleDeleteTicket(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input DeleteTicketInput,
) (*mcp.CallToolResult, DeleteTicketOutput, error) {
	if input.ID == "" {
		return nil, DeleteTicketOutput{}, NewValidationError("id", "cannot be empty")
	}

	err := s.store.Delete(input.ID)
	if err != nil {
		return nil, DeleteTicketOutput{}, WrapTicketError(err)
	}

	return nil, DeleteTicketOutput{
		Success: true,
		ID:      input.ID,
	}, nil
}

// handleMoveTicket moves a ticket to a different status.
func (s *Server) handleMoveTicket(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input MoveTicketInput,
) (*mcp.CallToolResult, MoveTicketOutput, error) {
	if input.ID == "" {
		return nil, MoveTicketOutput{}, NewValidationError("id", "cannot be empty")
	}
	if input.Status == "" {
		return nil, MoveTicketOutput{}, NewValidationError("status", "cannot be empty")
	}

	// Validate status
	status := ticket.Status(input.Status)
	if status != ticket.StatusBacklog && status != ticket.StatusProgress && status != ticket.StatusDone {
		return nil, MoveTicketOutput{}, NewValidationError("status", "must be backlog, progress, or done")
	}

	err := s.store.Move(input.ID, status)
	if err != nil {
		return nil, MoveTicketOutput{}, WrapTicketError(err)
	}

	return nil, MoveTicketOutput{
		Success: true,
		ID:      input.ID,
		Status:  input.Status,
	}, nil
}

// mcpServerConfig represents the MCP server configuration for claude.
type mcpServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

// claudeMCPConfig represents the claude MCP configuration file format.
type claudeMCPConfig struct {
	MCPServers map[string]mcpServerConfig `json:"mcpServers"`
}

// handleSpawnSession spawns a new agent session for a ticket.
func (s *Server) handleSpawnSession(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input SpawnSessionInput,
) (*mcp.CallToolResult, SpawnSessionOutput, error) {
	if input.TicketID == "" {
		return nil, SpawnSessionOutput{}, NewValidationError("ticket_id", "cannot be empty")
	}

	// Default agent to claude
	agent := input.Agent
	if agent == "" {
		agent = "claude"
	}

	// Validate ticket exists
	t, _, err := s.store.Get(input.TicketID)
	if err != nil {
		return nil, SpawnSessionOutput{}, WrapTicketError(err)
	}

	// Check if ticket already has active sessions
	if t.HasActiveSessions() {
		return nil, SpawnSessionOutput{
			Success:  false,
			TicketID: input.TicketID,
			Message:  "ticket already has an active session",
		}, nil
	}

	// Validate TmuxSession is configured
	if s.config.TmuxSession == "" {
		return nil, SpawnSessionOutput{
			Success: false,
			Message: "cannot spawn session: CORTEX_TMUX_SESSION not configured",
		}, nil
	}

	// Use injected tmux manager or create a new one
	tmuxMgr := s.tmuxManager
	if tmuxMgr == nil {
		var err error
		tmuxMgr, err = tmux.NewManager()
		if err != nil {
			return nil, SpawnSessionOutput{
				Success: false,
				Message: "tmux is not available: " + err.Error(),
			}, nil
		}
	}

	// Generate window name from ticket slug
	windowName := ticket.GenerateSlug(t.Title)

	// Add session to ticket store
	session, err := s.store.AddSession(input.TicketID, agent, windowName)
	if err != nil {
		return nil, SpawnSessionOutput{}, WrapTicketError(err)
	}

	// Find cortexd path - use injected path if provided (for testing)
	cortexdPath := s.config.CortexdPath
	if cortexdPath == "" {
		var err error
		cortexdPath, err = binpath.FindCortexd()
		if err != nil {
			// Cleanup session on failure
			_ = s.store.EndSession(input.TicketID, session.ID)
			return nil, SpawnSessionOutput{
				Success: false,
				Message: "cortexd not found: " + err.Error(),
			}, nil
		}
	}

	// Generate MCP config file
	mcpConfigPath := filepath.Join(os.TempDir(), fmt.Sprintf("cortex-mcp-%s.json", input.TicketID))
	mcpConfig := claudeMCPConfig{
		MCPServers: map[string]mcpServerConfig{
			"cortex": {
				Command: cortexdPath,
				Args:    []string{"mcp", "--ticket-id", input.TicketID},
				Env:     make(map[string]string),
			},
		},
	}

	// Add environment variables
	if s.config.TicketsDir != "" {
		mcpConfig.MCPServers["cortex"].Env["CORTEX_TICKETS_DIR"] = s.config.TicketsDir
	}
	if s.config.ProjectPath != "" {
		mcpConfig.MCPServers["cortex"].Env["CORTEX_PROJECT_PATH"] = s.config.ProjectPath
	}
	if s.config.TmuxSession != "" {
		mcpConfig.MCPServers["cortex"].Env["CORTEX_TMUX_SESSION"] = s.config.TmuxSession
	}

	mcpConfigData, err := json.MarshalIndent(mcpConfig, "", "  ")
	if err != nil {
		// Cleanup session on failure
		_ = s.store.EndSession(input.TicketID, session.ID)
		return nil, SpawnSessionOutput{}, NewInternalError("failed to marshal MCP config: " + err.Error())
	}

	if err := os.WriteFile(mcpConfigPath, mcpConfigData, 0644); err != nil {
		// Cleanup session on failure
		_ = s.store.EndSession(input.TicketID, session.ID)
		return nil, SpawnSessionOutput{}, NewInternalError("failed to write MCP config: " + err.Error())
	}

	// Build claude command with ticket prompt (like cortex0: uses permission-mode plan for tickets)
	// The agent can use the cortex MCP tools (readTicket, submitReport, approve) to interact
	promptText, err := prompt.LoadTicketAgent(s.config.ProjectPath, prompt.TicketVars{
		TicketID: input.TicketID,
		Title:    t.Title,
		Body:     t.Body,
		Slug:     ticket.GenerateSlug(t.Title),
	})
	if err != nil {
		// Cleanup session on failure
		_ = s.store.EndSession(input.TicketID, session.ID)
		_ = os.Remove(mcpConfigPath)
		return nil, SpawnSessionOutput{
			Success: false,
			Message: "failed to load ticket agent prompt: " + err.Error(),
		}, nil
	}
	// Use single quotes to prevent shell expansion (backticks, $vars, etc.)
	// Escape any single quotes in the prompt using POSIX pattern: ' -> '\''
	escapedPrompt := strings.ReplaceAll(promptText, "'", "'\\''")
	claudeCmd := fmt.Sprintf("claude '%s' --mcp-config %s --permission-mode plan", escapedPrompt, mcpConfigPath)

	// Spawn agent in tmux
	_, err = tmuxMgr.SpawnAgent(s.config.TmuxSession, windowName, claudeCmd)
	if err != nil {
		// Cleanup session and config on failure
		_ = s.store.EndSession(input.TicketID, session.ID)
		_ = os.Remove(mcpConfigPath)
		return nil, SpawnSessionOutput{
			Success: false,
			Message: "failed to spawn agent in tmux: " + err.Error(),
		}, nil
	}

	return nil, SpawnSessionOutput{
		Success:    true,
		TicketID:   input.TicketID,
		SessionID:  session.ID,
		TmuxWindow: windowName,
		Message:    fmt.Sprintf("Agent session spawned in tmux window '%s'", windowName),
	}, nil
}
