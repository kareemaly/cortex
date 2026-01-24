package mcp

import (
	"context"
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

	// Default agent to claude
	agent := input.Agent
	if agent == "" {
		agent = "claude"
	}

	// Default and validate mode
	mode := input.Mode
	if mode == "" {
		mode = "normal"
	}
	if mode != "normal" && mode != "resume" && mode != "fresh" {
		return nil, SpawnSessionOutput{}, NewValidationError("mode", "must be 'normal', 'resume', or 'fresh'")
	}

	// Validate ticket exists
	t, _, err := s.store.Get(input.TicketID)
	if err != nil {
		return nil, SpawnSessionOutput{}, WrapTicketError(err)
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
		tmuxMgr, err = tmux.NewManager()
		if err != nil {
			return nil, SpawnSessionOutput{
				Success: false,
				Message: "tmux is not available: " + err.Error(),
			}, nil
		}
	}

	// Detect current state
	stateInfo, err := spawn.DetectTicketState(t, s.config.TmuxSession, tmuxMgr)
	if err != nil {
		return nil, SpawnSessionOutput{}, WrapTicketError(err)
	}
	state := string(stateInfo.State)

	// Create spawner with dependencies
	spawner := spawn.NewSpawner(spawn.Dependencies{
		Store:       s.store,
		TmuxManager: tmuxMgr,
		CortexdPath: s.config.CortexdPath,
		Logger:      s.config.Logger,
	})

	// Determine if worktrees are enabled
	useWorktree := s.projectConfig != nil && s.projectConfig.Git.Worktrees

	// Execute based on state/mode matrix
	var result *spawn.SpawnResult
	switch stateInfo.State {
	case spawn.StateNormal:
		switch mode {
		case "normal":
			result, err = spawner.Spawn(spawn.SpawnRequest{
				AgentType:   spawn.AgentTypeTicketAgent,
				Agent:       agent,
				TmuxSession: s.config.TmuxSession,
				ProjectPath: s.config.ProjectPath,
				TicketsDir:  s.config.TicketsDir,
				TicketID:    input.TicketID,
				Ticket:      t,
				UseWorktree: useWorktree,
			})
		case "resume":
			return nil, SpawnSessionOutput{State: state}, NewStateConflictError(state, mode, "cannot resume - no existing session to resume")
		case "fresh":
			return nil, SpawnSessionOutput{State: state}, NewStateConflictError(state, mode, "cannot use fresh mode - no existing session to clear")
		}

	case spawn.StateActive:
		return nil, SpawnSessionOutput{State: state}, NewStateConflictError(state, mode, "session is currently active - wait for it to finish or close the tmux window")

	case spawn.StateOrphaned:
		switch mode {
		case "normal":
			return nil, SpawnSessionOutput{State: state}, NewStateConflictError(state, mode, "session was orphaned (tmux window closed). Use mode='resume' to continue or mode='fresh' to start over")
		case "resume":
			if stateInfo.Session == nil || stateInfo.Session.ID == "" {
				return nil, SpawnSessionOutput{State: state}, NewStateConflictError(state, mode, "cannot resume - no session ID stored")
			}
			result, err = spawner.Resume(spawn.ResumeRequest{
				AgentType:   spawn.AgentTypeTicketAgent,
				TmuxSession: s.config.TmuxSession,
				ProjectPath: s.config.ProjectPath,
				TicketsDir:  s.config.TicketsDir,
				SessionID:   stateInfo.Session.ID,
				WindowName:  stateInfo.Session.TmuxWindow,
				TicketID:    input.TicketID,
			})
		case "fresh":
			result, err = spawner.Fresh(spawn.SpawnRequest{
				AgentType:   spawn.AgentTypeTicketAgent,
				Agent:       agent,
				TmuxSession: s.config.TmuxSession,
				ProjectPath: s.config.ProjectPath,
				TicketsDir:  s.config.TicketsDir,
				TicketID:    input.TicketID,
				Ticket:      t,
				UseWorktree: useWorktree,
			})
		}

	case spawn.StateEnded:
		switch mode {
		case "normal":
			result, err = spawner.Spawn(spawn.SpawnRequest{
				AgentType:   spawn.AgentTypeTicketAgent,
				Agent:       agent,
				TmuxSession: s.config.TmuxSession,
				ProjectPath: s.config.ProjectPath,
				TicketsDir:  s.config.TicketsDir,
				TicketID:    input.TicketID,
				Ticket:      t,
				UseWorktree: useWorktree,
			})
		case "resume":
			return nil, SpawnSessionOutput{State: state}, NewStateConflictError(state, mode, "cannot resume - session has ended")
		case "fresh":
			result, err = spawner.Fresh(spawn.SpawnRequest{
				AgentType:   spawn.AgentTypeTicketAgent,
				Agent:       agent,
				TmuxSession: s.config.TmuxSession,
				ProjectPath: s.config.ProjectPath,
				TicketsDir:  s.config.TicketsDir,
				TicketID:    input.TicketID,
				Ticket:      t,
				UseWorktree: useWorktree,
			})
		}
	}

	if err != nil {
		if spawn.IsConfigError(err) || spawn.IsBinaryNotFoundError(err) {
			return nil, SpawnSessionOutput{
				Success: false,
				State:   state,
				Message: err.Error(),
			}, nil
		}
		return nil, SpawnSessionOutput{}, WrapTicketError(err)
	}

	return nil, SpawnSessionOutput{
		Success:    result.Success,
		TicketID:   input.TicketID,
		SessionID:  result.SessionID,
		TmuxWindow: result.TmuxWindow,
		State:      state,
		Message:    result.Message,
	}, nil
}
