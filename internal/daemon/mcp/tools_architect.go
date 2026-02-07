package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/daemon/api"
	"github.com/kareemaly/cortex/internal/install"
	"github.com/kareemaly/cortex/internal/types"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// validateProjectPath validates that a project path is registered and exists.
// Returns nil if projectPath is empty (uses default project).
// Returns a ToolError if the project is not registered or doesn't exist.
func (s *Server) validateProjectPath(projectPath string) *ToolError {
	if projectPath == "" {
		return nil // Use default project
	}

	// List all registered projects
	resp, err := s.sdkClient.ListProjects()
	if err != nil {
		return NewInternalError("failed to list projects: " + err.Error())
	}

	// Check if project is registered
	for _, p := range resp.Projects {
		if p.Path == projectPath {
			if !p.Exists {
				return NewValidationError("project_path", fmt.Sprintf("project directory does not exist: %s", projectPath))
			}
			return nil // Valid
		}
	}

	return NewValidationError("project_path", fmt.Sprintf("project not registered: %s (use 'cortex register %s' to register it)", projectPath, projectPath))
}

// getClientForProject returns an SDK client for the specified project.
// If projectPath is empty, returns the server's default client.
func (s *Server) getClientForProject(projectPath string) *sdk.Client {
	if projectPath == "" {
		return s.sdkClient
	}
	return s.sdkClient.WithProject(projectPath)
}

// registerArchitectTools registers all tools available to architect sessions.
func (s *Server) registerArchitectTools() {
	// List projects (cross-project)
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "listProjects",
		Description: "List all registered projects with their paths and titles. Use this to discover available projects for cross-project operations.",
	}, s.handleListProjects)

	// List tickets
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "listTickets",
		Description: "List tickets by status. Status parameter is required and must be one of: backlog, progress, review, done. Optionally specify project_path to list tickets from a different project.",
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

	// Add comment to a ticket
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "addTicketComment",
		Description: "Add a comment to a ticket (types: review_requested, done, blocker, comment)",
	}, s.handleArchitectAddComment)

	// Spawn session (stub)
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "spawnSession",
		Description: "Spawn a new agent session for a ticket",
	}, s.handleSpawnSession)

	// Get config docs
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "getCortexConfigDocs",
		Description: "Get CONFIG_DOCS.md with configuration documentation for this project's agent type",
	}, s.handleGetCortexConfigDocs)

	// Update due date
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "updateDueDate",
		Description: "Set or update the due date for a ticket",
	}, s.handleUpdateDueDate)

	// Clear due date
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "clearDueDate",
		Description: "Remove the due date from a ticket",
	}, s.handleClearDueDate)

	// Doc tools
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "createDoc",
		Description: "Create a new documentation file with markdown content and YAML frontmatter",
	}, s.handleCreateDoc)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "readDoc",
		Description: "Read a documentation file by ID",
	}, s.handleReadDoc)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "updateDoc",
		Description: "Update a documentation file's title, body, tags, or references",
	}, s.handleUpdateDoc)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "deleteDoc",
		Description: "Delete a documentation file by ID (current project only)",
	}, s.handleDeleteDoc)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "moveDoc",
		Description: "Move a documentation file to a different category/subdirectory",
	}, s.handleMoveDoc)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "listDocs",
		Description: "List documentation files with optional category, tag, and search filters",
	}, s.handleListDocs)

	// Add doc comment
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "addDocComment",
		Description: "Add a comment to a documentation file (types: review_requested, done, blocker, comment)",
	}, s.handleAddDocComment)

	// List sessions
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "listSessions",
		Description: "List all active agent sessions with ticket details",
	}, s.handleListSessions)
}

// handleListProjects lists all registered projects.
func (s *Server) handleListProjects(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ListProjectsInput,
) (*mcp.CallToolResult, ListProjectsOutput, error) {
	resp, err := s.sdkClient.ListProjects()
	if err != nil {
		return nil, ListProjectsOutput{}, wrapSDKError(err)
	}

	// Map SDK response to MCP output
	projects := make([]ProjectSummary, len(resp.Projects))
	for i, p := range resp.Projects {
		projects[i] = ProjectSummary{
			Path:   p.Path,
			Title:  p.Title,
			Exists: p.Exists,
		}
	}

	return nil, ListProjectsOutput{
		Projects: projects,
	}, nil
}

// handleListTickets lists tickets by status.
func (s *Server) handleListTickets(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ListTicketsInput,
) (*mcp.CallToolResult, ListTicketsOutput, error) {
	// Validate project path first
	if err := s.validateProjectPath(input.ProjectPath); err != nil {
		return nil, ListTicketsOutput{}, err
	}

	// Get client for target project
	client := s.getClientForProject(input.ProjectPath)

	// Validate status is provided and valid
	if input.Status == "" {
		return nil, ListTicketsOutput{}, NewValidationError("status", "is required")
	}
	if input.Status != "backlog" && input.Status != "progress" && input.Status != "review" && input.Status != "done" {
		return nil, ListTicketsOutput{}, NewValidationError("status", "must be one of: backlog, progress, review, done")
	}

	// Parse dueBefore if provided
	var dueBefore *time.Time
	if input.DueBefore != "" {
		parsed, err := time.Parse(time.RFC3339, input.DueBefore)
		if err != nil {
			return nil, ListTicketsOutput{}, NewValidationError("due_before", "must be in RFC3339 format")
		}
		dueBefore = &parsed
	}

	resp, err := client.ListTicketsByStatus(input.Status, input.Query, dueBefore, input.Tag)
	if err != nil {
		return nil, ListTicketsOutput{}, wrapSDKError(err)
	}

	// Map shared types to MCP-specific summaries
	summaries := make([]TicketSummary, len(resp.Tickets))
	for i, t := range resp.Tickets {
		summaries[i] = ticketSummaryResponseToMCP(&t)
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
	// Validate project path first
	if err := s.validateProjectPath(input.ProjectPath); err != nil {
		return nil, ReadTicketOutput{}, err
	}

	// Get client for target project
	client := s.getClientForProject(input.ProjectPath)

	if input.ID == "" {
		return nil, ReadTicketOutput{}, NewValidationError("id", "cannot be empty")
	}

	resp, err := client.GetTicketByID(input.ID)
	if err != nil {
		return nil, ReadTicketOutput{}, wrapSDKError(err)
	}

	return nil, ReadTicketOutput{
		Ticket: ticketResponseToOutput(resp),
	}, nil
}

// handleCreateTicket creates a new ticket via the daemon HTTP API.
func (s *Server) handleCreateTicket(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input CreateTicketInput,
) (*mcp.CallToolResult, CreateTicketOutput, error) {
	// Validate project path first
	if err := s.validateProjectPath(input.ProjectPath); err != nil {
		return nil, CreateTicketOutput{}, err
	}

	// Get client for target project
	client := s.getClientForProject(input.ProjectPath)

	// Parse dueDate if provided
	var dueDate *time.Time
	if input.DueDate != "" {
		parsed, err := time.Parse(time.RFC3339, input.DueDate)
		if err != nil {
			return nil, CreateTicketOutput{}, NewValidationError("due_date", "must be in RFC3339 format")
		}
		dueDate = &parsed
	}

	resp, err := client.CreateTicket(input.Title, input.Body, input.Type, dueDate, input.References, input.Tags)
	if err != nil {
		return nil, CreateTicketOutput{}, wrapSDKError(err)
	}

	return nil, CreateTicketOutput{
		Ticket: ticketResponseToOutput(resp),
	}, nil
}

// handleUpdateTicket updates a ticket's title and/or body via the daemon HTTP API.
func (s *Server) handleUpdateTicket(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input UpdateTicketInput,
) (*mcp.CallToolResult, UpdateTicketOutput, error) {
	// Validate project path first
	if err := s.validateProjectPath(input.ProjectPath); err != nil {
		return nil, UpdateTicketOutput{}, err
	}

	// Get client for target project
	client := s.getClientForProject(input.ProjectPath)

	if input.ID == "" {
		return nil, UpdateTicketOutput{}, NewValidationError("id", "cannot be empty")
	}

	resp, err := client.UpdateTicket(input.ID, input.Title, input.Body, input.References, input.Tags)
	if err != nil {
		return nil, UpdateTicketOutput{}, wrapSDKError(err)
	}

	return nil, UpdateTicketOutput{
		Ticket: ticketResponseToOutput(resp),
	}, nil
}

// handleDeleteTicket deletes a ticket via the daemon HTTP API.
func (s *Server) handleDeleteTicket(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input DeleteTicketInput,
) (*mcp.CallToolResult, DeleteTicketOutput, error) {
	if input.ID == "" {
		return nil, DeleteTicketOutput{}, NewValidationError("id", "cannot be empty")
	}

	err := s.sdkClient.DeleteTicket(input.ID)
	if err != nil {
		return nil, DeleteTicketOutput{}, wrapSDKError(err)
	}

	return nil, DeleteTicketOutput{
		Success: true,
		ID:      input.ID,
	}, nil
}

// handleMoveTicket moves a ticket to a different status via the daemon HTTP API.
func (s *Server) handleMoveTicket(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input MoveTicketInput,
) (*mcp.CallToolResult, MoveTicketOutput, error) {
	// Validate project path first
	if err := s.validateProjectPath(input.ProjectPath); err != nil {
		return nil, MoveTicketOutput{}, err
	}

	// Get client for target project
	client := s.getClientForProject(input.ProjectPath)

	if input.ID == "" {
		return nil, MoveTicketOutput{}, NewValidationError("id", "cannot be empty")
	}
	if input.Status == "" {
		return nil, MoveTicketOutput{}, NewValidationError("status", "cannot be empty")
	}

	// Validate status
	if input.Status != "backlog" && input.Status != "progress" && input.Status != "review" && input.Status != "done" {
		return nil, MoveTicketOutput{}, NewValidationError("status", "must be backlog, progress, review, or done")
	}

	_, err := client.MoveTicket(input.ID, input.Status)
	if err != nil {
		return nil, MoveTicketOutput{}, wrapSDKError(err)
	}

	return nil, MoveTicketOutput{
		Success: true,
		ID:      input.ID,
		Status:  input.Status,
	}, nil
}

// handleArchitectAddComment adds a comment to a ticket by ID.
func (s *Server) handleArchitectAddComment(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ArchitectAddCommentInput,
) (*mcp.CallToolResult, AddCommentOutput, error) {
	// Validate project path first
	if err := s.validateProjectPath(input.ProjectPath); err != nil {
		return nil, AddCommentOutput{}, err
	}

	// Get client for target project
	client := s.getClientForProject(input.ProjectPath)

	if input.ID == "" {
		return nil, AddCommentOutput{}, NewValidationError("id", "cannot be empty")
	}

	resp, err := client.AddComment(input.ID, input.Type, input.Content, "architect")
	if err != nil {
		return nil, AddCommentOutput{}, wrapSDKError(err)
	}

	return nil, AddCommentOutput{
		Success: resp.Success,
		Comment: resp.Comment,
	}, nil
}

// handleSpawnSession spawns a new agent session for a ticket.
// Delegates to the daemon HTTP API instead of calling spawn.Orchestrate() directly.
func (s *Server) handleSpawnSession(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input SpawnSessionInput,
) (*mcp.CallToolResult, SpawnSessionOutput, error) {
	// Validate project path first
	if err := s.validateProjectPath(input.ProjectPath); err != nil {
		return nil, SpawnSessionOutput{}, err
	}

	// Get client for target project
	client := s.getClientForProject(input.ProjectPath)

	// Determine which project path to use for the HTTP header
	projectPath := input.ProjectPath
	if projectPath == "" {
		projectPath = s.config.ProjectPath
	}

	if input.TicketID == "" {
		return nil, SpawnSessionOutput{}, NewValidationError("ticket_id", "cannot be empty")
	}

	// Validate mode if provided
	if input.Mode != "" && input.Mode != "normal" && input.Mode != "resume" && input.Mode != "fresh" {
		return nil, SpawnSessionOutput{}, NewValidationError("mode", "must be 'normal', 'resume', or 'fresh'")
	}

	// Look up ticket status via daemon API (needed to build the spawn URL)
	ticketResp, err := client.GetTicketByID(input.TicketID)
	if err != nil {
		return nil, SpawnSessionOutput{}, wrapSDKError(err)
	}

	// Build HTTP request to daemon
	url := fmt.Sprintf("%s/tickets/%s/%s/spawn", s.config.DaemonURL, ticketResp.Status, input.TicketID)
	if input.Mode != "" {
		url += "?mode=" + input.Mode
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, SpawnSessionOutput{}, NewInternalError("failed to create request: " + err.Error())
	}
	httpReq.Header.Set("X-Cortex-Project", projectPath)

	// Execute request
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, SpawnSessionOutput{}, NewInternalError("failed to contact daemon: " + err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	// Map response
	switch resp.StatusCode {
	case http.StatusCreated: // 201 - spawned/resumed
		var spawnResp api.SpawnResponse
		if err := json.NewDecoder(resp.Body).Decode(&spawnResp); err != nil {
			return nil, SpawnSessionOutput{}, NewInternalError("failed to decode response: " + err.Error())
		}
		return nil, SpawnSessionOutput{
			Success:    true,
			TicketID:   input.TicketID,
			TmuxWindow: spawnResp.Session.TmuxWindow,
		}, nil

	case http.StatusOK: // 200 - already active
		return nil, SpawnSessionOutput{State: "active"}, NewStateConflictError("active", input.Mode, "session is currently active - wait for it to finish or close the tmux window")

	case http.StatusConflict: // 409 - state/orphaned error
		var errResp types.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, SpawnSessionOutput{}, NewInternalError("failed to decode error response: " + err.Error())
		}
		state := parseStateFromError(errResp.Code, errResp.Error)
		return nil, SpawnSessionOutput{State: state}, NewStateConflictError(state, input.Mode, errResp.Error)

	case http.StatusNotFound: // 404 - ticket not found
		return nil, SpawnSessionOutput{}, NewNotFoundError("ticket", input.TicketID)

	case http.StatusBadRequest: // 400 - config error
		var errResp types.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, SpawnSessionOutput{}, NewInternalError("failed to decode error response: " + err.Error())
		}
		return nil, SpawnSessionOutput{
			Success: false,
			Message: errResp.Error,
		}, nil

	case http.StatusServiceUnavailable: // 503 - tmux unavailable
		var errResp types.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, SpawnSessionOutput{}, NewInternalError("failed to decode error response: " + err.Error())
		}
		return nil, SpawnSessionOutput{
			Success: false,
			Message: errResp.Error,
		}, nil

	default: // 500, etc.
		var errResp types.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, SpawnSessionOutput{
				Success: false,
				Message: fmt.Sprintf("daemon returned status %d", resp.StatusCode),
			}, nil
		}
		return nil, SpawnSessionOutput{
			Success: false,
			Message: errResp.Error,
		}, nil
	}
}

// parseStateFromError extracts the session state from an HTTP error response.
func parseStateFromError(code, message string) string {
	if code == "session_orphaned" {
		return "orphaned"
	}
	// For state_conflict, extract state from error message.
	// Format: "spawn: ticket <id> in state <state>: <message>"
	if idx := strings.Index(message, "in state "); idx != -1 {
		rest := message[idx+len("in state "):]
		if end := strings.Index(rest, ":"); end != -1 {
			return rest[:end]
		}
	}
	return "unknown"
}

// handleGetCortexConfigDocs returns the CONFIG_DOCS.md for this project's agent type.
func (s *Server) handleGetCortexConfigDocs(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetCortexConfigDocsInput,
) (*mcp.CallToolResult, GetCortexConfigDocsOutput, error) {
	// Determine config name from agent type
	// "claude" -> "claude-code" directory
	// "copilot" -> "copilot" directory (no "-code" suffix)
	configName := "claude-code" // default
	if s.projectConfig != nil && s.projectConfig.Architect.Agent != "" {
		agent := string(s.projectConfig.Architect.Agent)
		if agent == "copilot" {
			configName = "copilot"
		} else {
			configName = agent + "-code"
		}
	}

	// Try reading from resolved extend path first (allows user customization)
	if s.projectConfig != nil && s.projectConfig.ResolvedExtendPath() != "" {
		docPath := filepath.Join(s.projectConfig.ResolvedExtendPath(), "CONFIG_DOCS.md")
		if content, err := os.ReadFile(docPath); err == nil {
			return nil, GetCortexConfigDocsOutput{
				Content:    string(content),
				ConfigName: configName,
			}, nil
		}
	}

	// Fall back to embedded file
	content, err := install.GetConfigDocs(configName)
	if err != nil {
		return nil, GetCortexConfigDocsOutput{},
			NewNotFoundError("CONFIG_DOCS.md", configName)
	}

	return nil, GetCortexConfigDocsOutput{
		Content:    content,
		ConfigName: configName,
	}, nil
}

// handleUpdateDueDate sets or updates the due date for a ticket.
func (s *Server) handleUpdateDueDate(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input UpdateDueDateInput,
) (*mcp.CallToolResult, UpdateDueDateOutput, error) {
	// Validate project path first
	if err := s.validateProjectPath(input.ProjectPath); err != nil {
		return nil, UpdateDueDateOutput{}, err
	}

	// Get client for target project
	client := s.getClientForProject(input.ProjectPath)

	if input.ID == "" {
		return nil, UpdateDueDateOutput{}, NewValidationError("id", "cannot be empty")
	}
	if input.DueDate == "" {
		return nil, UpdateDueDateOutput{}, NewValidationError("due_date", "cannot be empty")
	}

	// Parse due date
	dueDate, err := time.Parse(time.RFC3339, input.DueDate)
	if err != nil {
		return nil, UpdateDueDateOutput{}, NewValidationError("due_date", "must be in RFC3339 format")
	}

	resp, err := client.SetDueDate(input.ID, dueDate)
	if err != nil {
		return nil, UpdateDueDateOutput{}, wrapSDKError(err)
	}

	return nil, UpdateDueDateOutput{
		Ticket: ticketResponseToOutput(resp),
	}, nil
}

// handleClearDueDate removes the due date from a ticket.
func (s *Server) handleClearDueDate(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ClearDueDateInput,
) (*mcp.CallToolResult, ClearDueDateOutput, error) {
	// Validate project path first
	if err := s.validateProjectPath(input.ProjectPath); err != nil {
		return nil, ClearDueDateOutput{}, err
	}

	// Get client for target project
	client := s.getClientForProject(input.ProjectPath)

	if input.ID == "" {
		return nil, ClearDueDateOutput{}, NewValidationError("id", "cannot be empty")
	}

	resp, err := client.ClearDueDate(input.ID)
	if err != nil {
		return nil, ClearDueDateOutput{}, wrapSDKError(err)
	}

	return nil, ClearDueDateOutput{
		Ticket: ticketResponseToOutput(resp),
	}, nil
}

// handleAddDocComment adds a comment to a doc by ID.
func (s *Server) handleAddDocComment(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input AddDocCommentInput,
) (*mcp.CallToolResult, AddCommentOutput, error) {
	if err := s.validateProjectPath(input.ProjectPath); err != nil {
		return nil, AddCommentOutput{}, err
	}

	client := s.getClientForProject(input.ProjectPath)

	if input.ID == "" {
		return nil, AddCommentOutput{}, NewValidationError("id", "cannot be empty")
	}
	if input.Content == "" {
		return nil, AddCommentOutput{}, NewValidationError("content", "cannot be empty")
	}

	resp, err := client.AddDocComment(input.ID, input.Type, input.Content, "architect")
	if err != nil {
		return nil, AddCommentOutput{}, wrapSDKError(err)
	}

	return nil, AddCommentOutput{
		Success: resp.Success,
		Comment: resp.Comment,
	}, nil
}

// handleListSessions lists all active sessions.
func (s *Server) handleListSessions(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ListSessionsInput,
) (*mcp.CallToolResult, ListSessionsOutput, error) {
	if err := s.validateProjectPath(input.ProjectPath); err != nil {
		return nil, ListSessionsOutput{}, err
	}

	client := s.getClientForProject(input.ProjectPath)

	resp, err := client.ListSessions()
	if err != nil {
		return nil, ListSessionsOutput{}, wrapSDKError(err)
	}

	items := make([]SessionListItem, len(resp.Sessions))
	for i, s := range resp.Sessions {
		items[i] = SessionListItem{
			SessionID:   s.SessionID,
			TicketID:    s.TicketID,
			TicketTitle: s.TicketTitle,
			Agent:       s.Agent,
			TmuxWindow:  s.TmuxWindow,
			StartedAt:   s.StartedAt,
			Status:      s.Status,
			Tool:        s.Tool,
		}
	}

	return nil, ListSessionsOutput{
		Sessions: items,
		Total:    len(items),
	}, nil
}
