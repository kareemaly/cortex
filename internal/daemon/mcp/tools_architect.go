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

// registerArchitectTools registers all tools available to architect sessions.
func (s *Server) registerArchitectTools() {
	// List tickets
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "listTickets",
		Description: "List tickets by status. Status parameter is required and must be one of: backlog, progress, done.",
	}, s.handleListTickets)

	// Read ticket
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "readTicket",
		Description: "Read full ticket details by ID",
	}, s.handleReadTicket)

	// Create work ticket
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "createWorkTicket",
		Description: "Create a new work ticket in backlog. Requires a repo field — the agent will spawn in that repo directory.",
	}, s.handleCreateWorkTicket)

	// Create research ticket
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "createResearchTicket",
		Description: "Create a new research ticket in backlog. No repo required — the agent spawns in the architect project root for read-only exploration.",
	}, s.handleCreateResearchTicket)

	// Update ticket
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "updateTicket",
		Description: "Update ticket title, body, and/or references",
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

	// List conclusions (persistent conclusion records)
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "listConclusions",
		Description: "List persistent conclusion records (metadata only, no body). Paginated, newest first. Use readConclusion to fetch full body.",
	}, s.handleListConclusions)

	// Read conclusion (persistent conclusion record)
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "readConclusion",
		Description: "Read a conclusion record by ID, including the full body.",
	}, s.handleReadConclusion)

	// Conclude architect session
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "concludeSession",
		Description: "Conclude the architect session and clean up",
	}, s.handleArchitectConcludeSession)
}

// handleListTickets lists tickets by status.
func (s *Server) handleListTickets(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ListTicketsInput,
) (*mcp.CallToolResult, ListTicketsOutput, error) {
	// Validate status is provided and valid
	if input.Status == "" {
		return nil, ListTicketsOutput{}, NewValidationError("status", "is required")
	}
	if input.Status != "backlog" && input.Status != "progress" && input.Status != "done" {
		return nil, ListTicketsOutput{}, NewValidationError("status", "must be one of: backlog, progress, done")
	}

	resp, err := s.sdkClient.ListTicketsByStatus(input.Status, input.Query, nil)
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
	if input.ID == "" {
		return nil, ReadTicketOutput{}, NewValidationError("id", "cannot be empty")
	}

	resp, err := s.sdkClient.GetTicketByID(input.ID)
	if err != nil {
		return nil, ReadTicketOutput{}, wrapSDKError(err)
	}

	return nil, ReadTicketOutput{
		Ticket: ticketResponseToOutput(resp),
	}, nil
}

// handleCreateWorkTicket creates a new work ticket via the daemon HTTP API.
func (s *Server) handleCreateWorkTicket(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input CreateWorkTicketInput,
) (*mcp.CallToolResult, CreateTicketOutput, error) {
	if input.Repo == "" {
		return nil, CreateTicketOutput{}, NewValidationError("repo", "is required for work tickets")
	}

	// Parse dueDate if provided
	var dueDate *time.Time
	if input.DueDate != "" {
		parsed, err := time.Parse(time.RFC3339, input.DueDate)
		if err != nil {
			return nil, CreateTicketOutput{}, NewValidationError("due_date", "must be in RFC3339 format")
		}
		dueDate = &parsed
	}

	resp, err := s.sdkClient.CreateTicket(input.Title, input.Body, "work", input.Repo, dueDate, input.References)
	if err != nil {
		return nil, CreateTicketOutput{}, wrapSDKError(err)
	}

	return nil, CreateTicketOutput{
		Ticket: ticketResponseToOutput(resp),
	}, nil
}

// handleCreateResearchTicket creates a new research ticket via the daemon HTTP API.
func (s *Server) handleCreateResearchTicket(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input CreateResearchTicketInput,
) (*mcp.CallToolResult, CreateTicketOutput, error) {
	// Parse dueDate if provided
	var dueDate *time.Time
	if input.DueDate != "" {
		parsed, err := time.Parse(time.RFC3339, input.DueDate)
		if err != nil {
			return nil, CreateTicketOutput{}, NewValidationError("due_date", "must be in RFC3339 format")
		}
		dueDate = &parsed
	}

	resp, err := s.sdkClient.CreateTicket(input.Title, input.Body, "research", "", dueDate, input.References)
	if err != nil {
		return nil, CreateTicketOutput{}, wrapSDKError(err)
	}

	return nil, CreateTicketOutput{
		Ticket: ticketResponseToOutput(resp),
	}, nil
}

// handleUpdateTicket updates a ticket's title, body, and/or references via the daemon HTTP API.
func (s *Server) handleUpdateTicket(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input UpdateTicketInput,
) (*mcp.CallToolResult, UpdateTicketOutput, error) {
	if input.ID == "" {
		return nil, UpdateTicketOutput{}, NewValidationError("id", "cannot be empty")
	}

	resp, err := s.sdkClient.UpdateTicket(input.ID, input.Title, input.Body, input.References)
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

	_, err := s.sdkClient.MoveTicket(input.ID, input.Status)
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
	if input.TicketID == "" {
		return nil, SpawnSessionOutput{}, NewValidationError("ticket_id", "cannot be empty")
	}

	// Validate mode if provided
	if input.Mode != "" && input.Mode != "normal" && input.Mode != "resume" && input.Mode != "fresh" {
		return nil, SpawnSessionOutput{}, NewValidationError("mode", "must be 'normal', 'resume', or 'fresh'")
	}

	projectPath := s.config.ArchitectPath

	// Look up ticket status via daemon API (needed to build the spawn URL)
	ticketResp, err := s.sdkClient.GetTicketByID(input.TicketID)
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
	httpReq.Header.Set("X-Cortex-Architect", projectPath)

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

	resp, err := s.sdkClient.SetDueDate(input.ID, dueDate)
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
	if input.ID == "" {
		return nil, ClearDueDateOutput{}, NewValidationError("id", "cannot be empty")
	}

	resp, err := s.sdkClient.ClearDueDate(input.ID)
	if err != nil {
		return nil, ClearDueDateOutput{}, wrapSDKError(err)
	}

	return nil, ClearDueDateOutput{
		Ticket: ticketResponseToOutput(resp),
	}, nil
}

// handleListConclusions lists persistent conclusion records (metadata only, no body).
func (s *Server) handleListConclusions(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ListConclusionsInput,
) (*mcp.CallToolResult, ListConclusionsOutput, error) {
	limit := input.Limit
	if limit == 0 {
		limit = 10
	}

	resp, err := s.sdkClient.ListConclusions(sdk.ListConclusionsParams{
		Type:   input.Type,
		Limit:  limit,
		Offset: input.Offset,
	})
	if err != nil {
		return nil, ListConclusionsOutput{}, wrapSDKError(err)
	}

	items := make([]ConclusionListItem, len(resp.Conclusions))
	for i, c := range resp.Conclusions {
		items[i] = ConclusionListItem{
			ID:      c.ID,
			Type:    c.Type,
			Ticket:  c.Ticket,
			Repo:    c.Repo,
			Created: c.Created.Format(time.RFC3339),
		}
	}

	return nil, ListConclusionsOutput{
		Conclusions: items,
		Total:       resp.Total,
	}, nil
}

// handleReadConclusion reads a conclusion record by ID, including full body.
func (s *Server) handleReadConclusion(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ReadConclusionInput,
) (*mcp.CallToolResult, ReadConclusionOutput, error) {
	if input.ID == "" {
		return nil, ReadConclusionOutput{}, NewValidationError("id", "cannot be empty")
	}

	resp, err := s.sdkClient.GetConclusion(input.ID)
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
