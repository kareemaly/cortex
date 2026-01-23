package mcp

import (
	"context"
	"log"

	"github.com/kareemaly/cortex/internal/lifecycle"
	"github.com/kareemaly/cortex/internal/ticket"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerTicketTools registers all tools available to ticket sessions.
func (s *Server) registerTicketTools() {
	// Read own ticket (no input needed, uses session ticket ID)
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "readTicket",
		Description: "Read your assigned ticket details",
	}, s.handleReadOwnTicket)

	// New move tools
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "moveTicketToProgress",
		Description: "Move your assigned ticket to in_progress status",
	}, s.handleMoveTicketToProgress)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "moveTicketToReview",
		Description: "Move your assigned ticket to review status",
	}, s.handleMoveTicketToReview)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "moveTicketToDone",
		Description: "Move your assigned ticket to done status. Optionally include a summary of completed work.",
	}, s.handleMoveTicketToDone)

	// Add comment tool
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "addTicketComment",
		Description: "Add a comment to your assigned ticket (types: scope_change, decision, blocker, progress, question, rejection, general, ticket_done)",
	}, s.handleAddTicketComment)
}

// EmptyInput is used for tools that don't require input.
type EmptyInput struct{}

// handleReadOwnTicket reads the ticket assigned to this session.
func (s *Server) handleReadOwnTicket(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input EmptyInput,
) (*mcp.CallToolResult, ReadTicketOutput, error) {
	t, status, err := s.store.Get(s.session.TicketID)
	if err != nil {
		return nil, ReadTicketOutput{}, WrapTicketError(err)
	}

	return nil, ReadTicketOutput{
		Ticket: ToTicketOutput(t, status),
	}, nil
}

// handleMoveTicketToProgress moves the assigned ticket to in_progress status.
func (s *Server) handleMoveTicketToProgress(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input EmptyInput,
) (*mcp.CallToolResult, PickupTicketOutput, error) {
	t, currentStatus, err := s.store.Get(s.session.TicketID)
	if err != nil {
		return nil, PickupTicketOutput{}, WrapTicketError(err)
	}

	if currentStatus == ticket.StatusProgress {
		return nil, PickupTicketOutput{
			Success: true,
			Message: "Ticket is already in progress",
		}, nil
	}

	if currentStatus == ticket.StatusDone {
		return nil, PickupTicketOutput{}, NewValidationError("status", "cannot move a completed ticket to progress")
	}

	err = s.store.Move(s.session.TicketID, ticket.StatusProgress)
	if err != nil {
		return nil, PickupTicketOutput{}, WrapTicketError(err)
	}

	// Execute moved_to_progress hooks
	var hooksOutput *HooksExecutionOutput
	hooks := s.getHooksForType(lifecycle.HookMovedToProgress)
	if len(hooks) > 0 && s.config.ProjectPath != "" {
		vars := buildTemplateVars(t)
		result, err := s.lifecycle.Execute(ctx, s.config.ProjectPath, lifecycle.HookMovedToProgress, hooks, vars)
		if err != nil {
			log.Printf("moved_to_progress hooks execution error: %v", err)
		} else {
			hooksOutput = convertExecutionResult(result)
		}
	}

	// Also run legacy on_pickup hooks for backward compatibility
	legacyHooks := s.getHooksForType(lifecycle.HookOnPickup)
	if len(legacyHooks) > 0 && s.config.ProjectPath != "" {
		vars := buildTemplateVars(t)
		result, err := s.lifecycle.Execute(ctx, s.config.ProjectPath, lifecycle.HookOnPickup, legacyHooks, vars)
		if err != nil {
			log.Printf("on_pickup hooks execution error: %v", err)
		} else if hooksOutput == nil {
			hooksOutput = convertExecutionResult(result)
		}
	}

	return nil, PickupTicketOutput{
		Success: true,
		Message: "Ticket moved to in_progress",
		Hooks:   hooksOutput,
	}, nil
}

// handleMoveTicketToReview moves the assigned ticket to review status.
func (s *Server) handleMoveTicketToReview(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input EmptyInput,
) (*mcp.CallToolResult, PickupTicketOutput, error) {
	t, currentStatus, err := s.store.Get(s.session.TicketID)
	if err != nil {
		return nil, PickupTicketOutput{}, WrapTicketError(err)
	}

	if currentStatus == ticket.StatusReview {
		return nil, PickupTicketOutput{
			Success: true,
			Message: "Ticket is already in review",
		}, nil
	}

	if currentStatus == ticket.StatusDone {
		return nil, PickupTicketOutput{}, NewValidationError("status", "cannot move a completed ticket to review")
	}

	err = s.store.Move(s.session.TicketID, ticket.StatusReview)
	if err != nil {
		return nil, PickupTicketOutput{}, WrapTicketError(err)
	}

	// Execute moved_to_review hooks
	var hooksOutput *HooksExecutionOutput
	hooks := s.getHooksForType(lifecycle.HookMovedToReview)
	if len(hooks) > 0 && s.config.ProjectPath != "" {
		vars := buildTemplateVars(t)
		result, err := s.lifecycle.Execute(ctx, s.config.ProjectPath, lifecycle.HookMovedToReview, hooks, vars)
		if err != nil {
			log.Printf("moved_to_review hooks execution error: %v", err)
		} else {
			hooksOutput = convertExecutionResult(result)
		}
	}

	return nil, PickupTicketOutput{
		Success: true,
		Message: "Ticket moved to review",
		Hooks:   hooksOutput,
	}, nil
}

// handleMoveTicketToDone moves the assigned ticket to done status.
func (s *Server) handleMoveTicketToDone(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input MoveTicketToDoneInput,
) (*mcp.CallToolResult, ApproveOutput, error) {
	t, currentStatus, err := s.store.Get(s.session.TicketID)
	if err != nil {
		return nil, ApproveOutput{}, WrapTicketError(err)
	}

	if currentStatus == ticket.StatusDone {
		return nil, ApproveOutput{
			Success:  true,
			TicketID: s.session.TicketID,
			Status:   string(ticket.StatusDone),
			Message:  "Ticket is already done",
		}, nil
	}

	// Add summary as ticket_done comment if provided
	if input.Summary != "" {
		var sessionID string
		if t.Session != nil && t.Session.IsActive() {
			sessionID = t.Session.ID
		}
		_, err = s.store.AddComment(s.session.TicketID, sessionID, ticket.CommentTicketDone, input.Summary)
		if err != nil {
			return nil, ApproveOutput{}, WrapTicketError(err)
		}
	}

	// Execute moved_to_done hooks - MUST succeed before moving
	var hooksOutput *HooksExecutionOutput
	hooks := s.getHooksForType(lifecycle.HookMovedToDone)
	if len(hooks) > 0 && s.config.ProjectPath != "" {
		vars := buildTemplateVars(t)
		result, err := s.lifecycle.Execute(ctx, s.config.ProjectPath, lifecycle.HookMovedToDone, hooks, vars)
		if err != nil {
			log.Printf("moved_to_done hooks execution error: %v", err)
			return nil, ApproveOutput{
				Success:  false,
				TicketID: s.session.TicketID,
				Status:   string(currentStatus),
				Message:  "moved_to_done hooks failed to execute: " + err.Error(),
			}, nil
		}
		hooksOutput = convertExecutionResult(result)
		if !result.Success {
			return nil, ApproveOutput{
				Success:  false,
				TicketID: s.session.TicketID,
				Status:   string(currentStatus),
				Message:  "moved_to_done hooks failed (non-zero exit code)",
				Hooks:    hooksOutput,
			}, nil
		}
	}

	err = s.store.Move(s.session.TicketID, ticket.StatusDone)
	if err != nil {
		return nil, ApproveOutput{}, WrapTicketError(err)
	}

	return nil, ApproveOutput{
		Success:  true,
		TicketID: s.session.TicketID,
		Status:   string(ticket.StatusDone),
		Hooks:    hooksOutput,
	}, nil
}

// handleAddTicketComment adds a comment to the assigned ticket.
func (s *Server) handleAddTicketComment(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input AddCommentInput,
) (*mcp.CallToolResult, AddCommentOutput, error) {
	t, _, err := s.store.Get(s.session.TicketID)
	if err != nil {
		return nil, AddCommentOutput{}, WrapTicketError(err)
	}

	// Find active session ID
	var activeSessionID string
	if t.Session != nil && t.Session.IsActive() {
		activeSessionID = t.Session.ID
	}

	// Validate comment type
	commentType := ticket.CommentType(input.Type)
	switch commentType {
	case ticket.CommentScopeChange, ticket.CommentDecision, ticket.CommentBlocker,
		ticket.CommentProgress, ticket.CommentQuestion, ticket.CommentRejection,
		ticket.CommentGeneral, ticket.CommentTicketDone:
		// Valid type
	default:
		return nil, AddCommentOutput{}, NewValidationError("type", "invalid comment type")
	}

	comment, err := s.store.AddComment(s.session.TicketID, activeSessionID, commentType, input.Content)
	if err != nil {
		return nil, AddCommentOutput{}, WrapTicketError(err)
	}

	// Execute comment_added hooks
	var hooksOutput *HooksExecutionOutput
	hooks := s.getHooksForType(lifecycle.HookCommentAdded)
	if len(hooks) > 0 && s.config.ProjectPath != "" {
		vars := buildTemplateVars(t).WithComment(input.Type, input.Content)
		if activeSessionID != "" {
			vars = vars.WithSession(activeSessionID, "")
		}
		result, err := s.lifecycle.Execute(ctx, s.config.ProjectPath, lifecycle.HookCommentAdded, hooks, vars)
		if err != nil {
			log.Printf("comment_added hooks execution error: %v", err)
		} else {
			hooksOutput = convertExecutionResult(result)
		}
	}

	return nil, AddCommentOutput{
		Success: true,
		Comment: CommentOutput{
			ID:        comment.ID,
			SessionID: comment.SessionID,
			Type:      string(comment.Type),
			Content:   comment.Content,
			CreatedAt: comment.CreatedAt,
		},
		Hooks: hooksOutput,
	}, nil
}
