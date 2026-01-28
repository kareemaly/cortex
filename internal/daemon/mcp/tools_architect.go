package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/kareemaly/cortex/internal/daemon/api"
	"github.com/kareemaly/cortex/internal/types"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerArchitectTools registers all tools available to architect sessions.
func (s *Server) registerArchitectTools() {
	// List tickets
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "listTickets",
		Description: "List tickets by status. Status parameter is required and must be one of: backlog, progress, review, done",
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
	if input.Status != "backlog" && input.Status != "progress" && input.Status != "review" && input.Status != "done" {
		return nil, ListTicketsOutput{}, NewValidationError("status", "must be one of: backlog, progress, review, done")
	}

	resp, err := s.sdkClient.ListTicketsByStatus(input.Status, input.Query)
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

// handleCreateTicket creates a new ticket via the daemon HTTP API.
func (s *Server) handleCreateTicket(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input CreateTicketInput,
) (*mcp.CallToolResult, CreateTicketOutput, error) {
	resp, err := s.sdkClient.CreateTicket(input.Title, input.Body, input.Type)
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
	if input.ID == "" {
		return nil, UpdateTicketOutput{}, NewValidationError("id", "cannot be empty")
	}

	resp, err := s.sdkClient.UpdateTicket(input.ID, input.Title, input.Body)
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
	if input.Status != "backlog" && input.Status != "progress" && input.Status != "review" && input.Status != "done" {
		return nil, MoveTicketOutput{}, NewValidationError("status", "must be backlog, progress, review, or done")
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

// handleArchitectAddComment adds a comment to a ticket by ID.
func (s *Server) handleArchitectAddComment(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ArchitectAddCommentInput,
) (*mcp.CallToolResult, AddCommentOutput, error) {
	if input.ID == "" {
		return nil, AddCommentOutput{}, NewValidationError("id", "cannot be empty")
	}

	resp, err := s.sdkClient.AddComment(input.ID, input.Type, input.Content)
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
	if input.TicketID == "" {
		return nil, SpawnSessionOutput{}, NewValidationError("ticket_id", "cannot be empty")
	}

	// Validate mode if provided
	if input.Mode != "" && input.Mode != "normal" && input.Mode != "resume" && input.Mode != "fresh" {
		return nil, SpawnSessionOutput{}, NewValidationError("mode", "must be 'normal', 'resume', or 'fresh'")
	}

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
	httpReq.Header.Set("X-Cortex-Project", s.config.ProjectPath)

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
			SessionID:  spawnResp.Session.ID,
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
