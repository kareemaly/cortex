package mcp

import (
	"context"
	"strings"
	"time"

	"github.com/kareemaly/cortex1/internal/ticket"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerArchitectTools registers all tools available to architect sessions.
func (s *Server) registerArchitectTools() {
	// List tickets
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "listTickets",
		Description: "List tickets with optional status filter",
	}, s.handleListTickets)

	// Search tickets
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "searchTickets",
		Description: "Search tickets by title/body with optional date filters",
	}, s.handleSearchTickets)

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

	// Get session status
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "getSessionStatus",
		Description: "Get the status of an active session",
	}, s.handleGetSessionStatus)
}

// handleListTickets lists tickets with optional status filter.
func (s *Server) handleListTickets(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ListTicketsInput,
) (*mcp.CallToolResult, ListTicketsOutput, error) {
	var summaries []TicketSummary

	if input.Status != "" {
		// List by specific status
		status := ticket.Status(input.Status)
		tickets, err := s.store.List(status)
		if err != nil {
			return nil, ListTicketsOutput{}, WrapTicketError(err)
		}
		for _, t := range tickets {
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
				summaries = append(summaries, ToTicketSummary(t, status))
			}
		}
	}

	return nil, ListTicketsOutput{
		Tickets: summaries,
		Total:   len(summaries),
	}, nil
}

// handleSearchTickets searches tickets by title/body with optional date filters.
func (s *Server) handleSearchTickets(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input SearchTicketsInput,
) (*mcp.CallToolResult, ListTicketsOutput, error) {
	if input.Query == "" {
		return nil, ListTicketsOutput{}, NewValidationError("query", "cannot be empty")
	}

	// Parse optional date filters
	var fromDate, toDate *time.Time
	if input.FromDate != "" {
		t, err := time.Parse(time.RFC3339, input.FromDate)
		if err != nil {
			return nil, ListTicketsOutput{}, NewValidationError("from_date", "invalid RFC3339 format")
		}
		fromDate = &t
	}
	if input.ToDate != "" {
		t, err := time.Parse(time.RFC3339, input.ToDate)
		if err != nil {
			return nil, ListTicketsOutput{}, NewValidationError("to_date", "invalid RFC3339 format")
		}
		toDate = &t
	}

	// Get all tickets and filter
	allTickets, err := s.store.ListAll()
	if err != nil {
		return nil, ListTicketsOutput{}, WrapTicketError(err)
	}

	query := strings.ToLower(input.Query)
	var summaries []TicketSummary

	for status, tickets := range allTickets {
		for _, t := range tickets {
			// Check query match
			if !strings.Contains(strings.ToLower(t.Title), query) &&
				!strings.Contains(strings.ToLower(t.Body), query) {
				continue
			}

			// Check date filters
			if fromDate != nil && t.Dates.Created.Before(*fromDate) {
				continue
			}
			if toDate != nil && t.Dates.Created.After(*toDate) {
				continue
			}

			summaries = append(summaries, ToTicketSummary(t, status))
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

// handleSpawnSession is a stub that returns "not implemented".
func (s *Server) handleSpawnSession(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input SpawnSessionInput,
) (*mcp.CallToolResult, SpawnSessionOutput, error) {
	return nil, SpawnSessionOutput{
		Message: "spawnSession is not implemented yet. Tmux integration is tracked in a separate ticket.",
	}, nil
}

// handleGetSessionStatus gets the status of an active session.
func (s *Server) handleGetSessionStatus(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetSessionStatusInput,
) (*mcp.CallToolResult, GetSessionStatusOutput, error) {
	if input.TicketID == "" {
		return nil, GetSessionStatusOutput{}, NewValidationError("ticket_id", "cannot be empty")
	}

	t, _, err := s.store.Get(input.TicketID)
	if err != nil {
		return nil, GetSessionStatusOutput{}, WrapTicketError(err)
	}

	// Find the session
	var session *ticket.Session
	if input.SessionID != "" {
		// Find specific session
		for i := range t.Sessions {
			if t.Sessions[i].ID == input.SessionID {
				session = &t.Sessions[i]
				break
			}
		}
		if session == nil {
			return nil, GetSessionStatusOutput{}, NewNotFoundError("session", input.SessionID)
		}
	} else {
		// Find active session
		for i := range t.Sessions {
			if t.Sessions[i].IsActive() {
				session = &t.Sessions[i]
				break
			}
		}
		if session == nil {
			return nil, GetSessionStatusOutput{
				Message: "No active session found for this ticket",
			}, nil
		}
	}

	output := ToSessionOutput(session)
	return nil, GetSessionStatusOutput{
		Session: &output,
	}, nil
}
