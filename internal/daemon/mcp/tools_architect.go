package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/daemon/api"
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
		Description: "List tickets by status. Status parameter is required and must be one of: backlog, progress, done. Optionally specify project_path to list tickets from a different project.",
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
		Description: "Update ticket title, body, type, references, and/or tags",
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

	// Spawn session
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "spawnSession",
		Description: "Spawn a new agent session for a ticket",
	}, s.handleSpawnSession)

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

	// Note tools
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "listNotes",
		Description: "List all active project notes/reminders",
	}, s.handleListNotes)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "createNote",
		Description: "Create a new note/reminder with optional due date (YYYY-MM-DD)",
	}, s.handleCreateNote)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "updateNote",
		Description: "Update a note's text and/or due date",
	}, s.handleUpdateNote)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "deleteNote",
		Description: "Delete a note by ID",
	}, s.handleDeleteNote)

	// List sessions (persistent conclusions)
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "listSessions",
		Description: "List all persistent session conclusions",
	}, s.handleListSessions)

	// Read session (persistent conclusion)
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "readSession",
		Description: "Read a persistent session/conclusion record by ID",
	}, s.handleReadSession)

	// Conclude architect session
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "concludeSession",
		Description: "Conclude the architect session and clean up",
	}, s.handleArchitectConcludeSession)
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
	if input.Status != "backlog" && input.Status != "progress" && input.Status != "done" {
		return nil, ListTicketsOutput{}, NewValidationError("status", "must be one of: backlog, progress, done")
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

	resp, err := client.UpdateTicket(input.ID, input.Title, input.Body, input.Type, input.References, input.Tags)
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
	if input.Status != "backlog" && input.Status != "progress" && input.Status != "done" {
		return nil, MoveTicketOutput{}, NewValidationError("status", "must be backlog, progress, or done")
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

// handleListSessions lists all persistent session conclusions.
func (s *Server) handleListSessions(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ListConclusionsInput,
) (*mcp.CallToolResult, ListConclusionsOutput, error) {
	if err := s.validateProjectPath(input.ProjectPath); err != nil {
		return nil, ListConclusionsOutput{}, err
	}

	client := s.getClientForProject(input.ProjectPath)

	resp, err := client.ListConclusions()
	if err != nil {
		return nil, ListConclusionsOutput{}, wrapSDKError(err)
	}

	items := make([]ConclusionOutput, len(resp.Conclusions))
	for i, c := range resp.Conclusions {
		items[i] = ConclusionOutput{
			ID:      c.ID,
			Type:    c.Type,
			Ticket:  c.Ticket,
			Repo:    c.Repo,
			Body:    c.Body,
			Created: c.Created.Format(time.RFC3339),
		}
	}

	return nil, ListConclusionsOutput{
		Conclusions: items,
		Total:       len(items),
	}, nil
}

// handleReadSession reads a persistent session/conclusion record by ID.
func (s *Server) handleReadSession(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ReadConclusionInput,
) (*mcp.CallToolResult, ReadConclusionOutput, error) {
	if input.ID == "" {
		return nil, ReadConclusionOutput{}, NewValidationError("id", "cannot be empty")
	}

	if err := s.validateProjectPath(input.ProjectPath); err != nil {
		return nil, ReadConclusionOutput{}, err
	}

	client := s.getClientForProject(input.ProjectPath)

	resp, err := client.GetConclusion(input.ID)
	if err != nil {
		return nil, ReadConclusionOutput{}, wrapSDKError(err)
	}

	return nil, ReadConclusionOutput{
		Conclusion: ConclusionOutput{
			ID:      resp.ID,
			Type:    resp.Type,
			Ticket:  resp.Ticket,
			Repo:    resp.Repo,
			Body:    resp.Body,
			Created: resp.Created.Format(time.RFC3339),
		},
	}, nil
}

// handleArchitectConcludeSession concludes the architect session.
func (s *Server) handleArchitectConcludeSession(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ConcludeSessionInput,
) (*mcp.CallToolResult, ArchitectConcludeOutput, error) {
	if input.Content == "" {
		return nil, ArchitectConcludeOutput{}, NewValidationError("content", "cannot be empty")
	}

	resp, err := s.sdkClient.ConcludeArchitectSession(input.Content)
	if err != nil {
		return nil, ArchitectConcludeOutput{}, wrapSDKError(err)
	}

	return nil, ArchitectConcludeOutput{
		Success: resp.Success,
		Message: resp.Message,
	}, nil
}
