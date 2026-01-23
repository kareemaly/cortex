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
		Description: "Move your assigned ticket to done status",
	}, s.handleMoveTicketToDone)

	// Add comment tool
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "addTicketComment",
		Description: "Add a comment to your assigned ticket (types: scope_change, decision, blocker, progress, question, rejection, general)",
	}, s.handleAddTicketComment)

	// Conclude session tool
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "concludeSession",
		Description: "End your current session without moving the ticket",
	}, s.handleConcludeSession)

	// Deprecated tools (kept for backward compatibility)
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "pickupTicket",
		Description: "[DEPRECATED: Use moveTicketToProgress] Start working on your assigned ticket",
	}, s.handlePickupTicket)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "submitReport",
		Description: "[DEPRECATED: Use addTicketComment] Update your session report",
	}, s.handleSubmitReport)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "approve",
		Description: "[DEPRECATED: Use moveTicketToDone] Mark your work as complete",
	}, s.handleApprove)
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
	input EmptyInput,
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
		ticket.CommentProgress, ticket.CommentQuestion, ticket.CommentRejection, ticket.CommentGeneral:
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

// handleConcludeSession ends the active session without moving the ticket.
func (s *Server) handleConcludeSession(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ConcludeSessionInput,
) (*mcp.CallToolResult, ConcludeSessionOutput, error) {
	t, _, err := s.store.Get(s.session.TicketID)
	if err != nil {
		return nil, ConcludeSessionOutput{}, WrapTicketError(err)
	}

	// Get active session
	if t.Session == nil || !t.Session.IsActive() {
		return nil, ConcludeSessionOutput{}, NewValidationError("session", "no active session found")
	}
	activeSessionID := t.Session.ID

	// Save session reference before ending (for tmux window cleanup)
	sessionToClose := t.Session

	// Add summary as a comment if provided
	if input.Summary != "" {
		_, err = s.store.AddComment(s.session.TicketID, activeSessionID, ticket.CommentProgress, input.Summary)
		if err != nil {
			return nil, ConcludeSessionOutput{}, WrapTicketError(err)
		}
	}

	// End the session
	err = s.store.EndSession(s.session.TicketID)
	if err != nil {
		return nil, ConcludeSessionOutput{}, WrapTicketError(err)
	}

	// Execute session_ended hooks
	var hooksOutput *HooksExecutionOutput
	hooks := s.getHooksForType(lifecycle.HookSessionEnded)
	if len(hooks) > 0 && s.config.ProjectPath != "" {
		vars := buildTemplateVars(t).WithSession(activeSessionID, "")
		result, err := s.lifecycle.Execute(ctx, s.config.ProjectPath, lifecycle.HookSessionEnded, hooks, vars)
		if err != nil {
			log.Printf("session_ended hooks execution error: %v", err)
		} else {
			hooksOutput = convertExecutionResult(result)
		}
	}

	// Kill tmux window (best-effort cleanup)
	s.killSessionWindow(sessionToClose)

	return nil, ConcludeSessionOutput{
		Success: true,
		Message: "Session ended successfully",
		Hooks:   hooksOutput,
	}, nil
}

// --- Deprecated handlers (for backward compatibility) ---

// handlePickupTicket is deprecated - use handleMoveTicketToProgress instead.
func (s *Server) handlePickupTicket(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input EmptyInput,
) (*mcp.CallToolResult, PickupTicketOutput, error) {
	return s.handleMoveTicketToProgress(ctx, req, input)
}

// handleSubmitReport is deprecated - use handleAddTicketComment instead.
func (s *Server) handleSubmitReport(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input SubmitReportInput,
) (*mcp.CallToolResult, SubmitReportOutput, error) {
	t, _, err := s.store.Get(s.session.TicketID)
	if err != nil {
		return nil, SubmitReportOutput{}, WrapTicketError(err)
	}

	// Get active session
	if t.Session == nil || !t.Session.IsActive() {
		return nil, SubmitReportOutput{}, NewValidationError("session", "no active session found")
	}
	activeSessionID := t.Session.ID

	// Convert to comments
	if input.ScopeChanges != nil && *input.ScopeChanges != "" {
		_, err = s.store.AddComment(s.session.TicketID, activeSessionID, ticket.CommentScopeChange, *input.ScopeChanges)
		if err != nil {
			return nil, SubmitReportOutput{}, WrapTicketError(err)
		}
	}

	for _, decision := range input.Decisions {
		_, err = s.store.AddComment(s.session.TicketID, activeSessionID, ticket.CommentDecision, decision)
		if err != nil {
			return nil, SubmitReportOutput{}, WrapTicketError(err)
		}
	}

	if input.Summary != "" {
		_, err = s.store.AddComment(s.session.TicketID, activeSessionID, ticket.CommentProgress, input.Summary)
		if err != nil {
			return nil, SubmitReportOutput{}, WrapTicketError(err)
		}
	}

	// Execute on_submit hooks for backward compatibility
	var hooksOutput *HooksExecutionOutput
	hooks := s.getHooksForType(lifecycle.HookOnSubmit)
	if len(hooks) > 0 && s.config.ProjectPath != "" {
		vars := buildTemplateVars(t)
		result, err := s.lifecycle.Execute(ctx, s.config.ProjectPath, lifecycle.HookOnSubmit, hooks, vars)
		if err != nil {
			log.Printf("on_submit hooks execution error: %v", err)
		} else {
			hooksOutput = convertExecutionResult(result)
		}
	}

	return nil, SubmitReportOutput{
		Success: true,
		Message: "Report submitted as comments",
		Hooks:   hooksOutput,
	}, nil
}

// handleApprove is deprecated - use handleMoveTicketToDone instead.
func (s *Server) handleApprove(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ApproveInput,
) (*mcp.CallToolResult, ApproveOutput, error) {
	t, currentStatus, err := s.store.Get(s.session.TicketID)
	if err != nil {
		return nil, ApproveOutput{}, WrapTicketError(err)
	}

	// Get active session
	if t.Session == nil || !t.Session.IsActive() {
		return nil, ApproveOutput{}, NewValidationError("session", "no active session found")
	}
	activeSessionID := t.Session.ID

	// Save session reference before ending (for tmux window cleanup)
	sessionToClose := t.Session

	// Add summary as comment if provided
	if input.Summary != "" {
		_, err = s.store.AddComment(s.session.TicketID, activeSessionID, ticket.CommentProgress, input.Summary)
		if err != nil {
			return nil, ApproveOutput{}, WrapTicketError(err)
		}
	}

	// Execute on_approve hooks - MUST succeed before moving to done
	var hooksOutput *HooksExecutionOutput
	hooks := s.getHooksForType(lifecycle.HookOnApprove)
	if len(hooks) > 0 && s.config.ProjectPath != "" {
		vars := buildTemplateVars(t).WithCommitMessage(input.CommitMessage)
		result, err := s.lifecycle.Execute(ctx, s.config.ProjectPath, lifecycle.HookOnApprove, hooks, vars)
		if err != nil {
			log.Printf("on_approve hooks execution error: %v", err)
			return nil, ApproveOutput{
				Success:  false,
				TicketID: s.session.TicketID,
				Status:   string(currentStatus),
				Message:  "on_approve hooks failed to execute: " + err.Error(),
			}, nil
		}
		hooksOutput = convertExecutionResult(result)
		if !result.Success {
			return nil, ApproveOutput{
				Success:  false,
				TicketID: s.session.TicketID,
				Status:   string(currentStatus),
				Message:  "on_approve hooks failed (non-zero exit code)",
				Hooks:    hooksOutput,
			}, nil
		}
	}

	// End the session
	err = s.store.EndSession(s.session.TicketID)
	if err != nil {
		return nil, ApproveOutput{}, WrapTicketError(err)
	}

	// Move ticket to done
	err = s.store.Move(s.session.TicketID, ticket.StatusDone)
	if err != nil {
		return nil, ApproveOutput{}, WrapTicketError(err)
	}

	// Kill tmux window (best-effort cleanup)
	s.killSessionWindow(sessionToClose)

	return nil, ApproveOutput{
		Success:  true,
		TicketID: s.session.TicketID,
		Status:   string(ticket.StatusDone),
		Hooks:    hooksOutput,
	}, nil
}
