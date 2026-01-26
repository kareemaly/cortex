package mcp

import (
	"context"
	"errors"
	"strings"

	"github.com/kareemaly/cortex/internal/core/spawn"
	"github.com/kareemaly/cortex/internal/ticket"
	"github.com/kareemaly/cortex/internal/tmux"
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
	status := ticket.Status(input.Status)
	if status != ticket.StatusBacklog && status != ticket.StatusProgress && status != ticket.StatusReview && status != ticket.StatusDone {
		return nil, ListTicketsOutput{}, NewValidationError("status", "must be one of: backlog, progress, review, done")
	}

	// Initialize as empty slice (not nil) to ensure JSON marshals to [] not null
	summaries := []TicketSummary{}

	// Prepare query for case-insensitive matching
	query := strings.ToLower(input.Query)

	// List by status
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

// handleSpawnSession spawns a new agent session for a ticket.
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

	// Delegate to shared orchestration
	result, err := spawn.Orchestrate(ctx, spawn.OrchestrateRequest{
		TicketID:    input.TicketID,
		Mode:        input.Mode,
		Agent:       input.Agent,
		ProjectPath: s.config.ProjectPath,
		TicketsDir:  s.config.TicketsDir,
		TmuxSession: s.config.TmuxSession,
	}, spawn.OrchestrateDeps{
		Store:       s.store,
		TmuxManager: tmuxMgr,
		CortexdPath: s.config.CortexdPath,
		Logger:      s.config.Logger,
	})
	if err != nil {
		var stateErr *spawn.StateError
		if errors.As(err, &stateErr) {
			return nil, SpawnSessionOutput{State: string(stateErr.State)}, NewStateConflictError(string(stateErr.State), input.Mode, stateErr.Message)
		}
		if spawn.IsConfigError(err) || spawn.IsBinaryNotFoundError(err) {
			return nil, SpawnSessionOutput{
				Success: false,
				Message: err.Error(),
			}, nil
		}
		return nil, SpawnSessionOutput{}, WrapTicketError(err)
	}

	// Already active: return state conflict
	if result.Outcome == spawn.OutcomeAlreadyActive {
		state := string(result.StateInfo.State)
		return nil, SpawnSessionOutput{State: state}, NewStateConflictError(state, input.Mode, "session is currently active - wait for it to finish or close the tmux window")
	}

	// Spawned or resumed
	return nil, SpawnSessionOutput{
		Success:    true,
		TicketID:   input.TicketID,
		SessionID:  result.SpawnResult.SessionID,
		TmuxWindow: result.SpawnResult.TmuxWindow,
		State:      string(result.StateInfo.State),
		Message:    result.SpawnResult.Message,
	}, nil
}
