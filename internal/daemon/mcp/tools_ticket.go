package mcp

import (
	"context"
	"log"

	"github.com/kareemaly/cortex/internal/ticket"
	"github.com/kareemaly/cortex/internal/worktree"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerTicketTools registers all tools available to ticket sessions.
func (s *Server) registerTicketTools() {
	// Read own ticket (no input needed, uses session ticket ID)
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "readTicket",
		Description: "Read your assigned ticket details",
	}, s.handleReadOwnTicket)

	// Add comment tool
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "addTicketComment",
		Description: "Add a comment to your assigned ticket (types: scope_change, decision, blocker, progress, question, rejection, general, ticket_done)",
	}, s.handleAddTicketComment)

	// Request review tool
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "requestReview",
		Description: "Request a human review for a repository. Call this after committing changes to request approval.",
	}, s.handleRequestReview)

	// Conclude session tool
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "concludeSession",
		Description: "Conclude the session and mark the ticket as done. Call this after all reviews are approved.",
	}, s.handleConcludeSession)
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

	return nil, AddCommentOutput{
		Success: true,
		Comment: CommentOutput{
			ID:        comment.ID,
			SessionID: comment.SessionID,
			Type:      string(comment.Type),
			Content:   comment.Content,
			CreatedAt: comment.CreatedAt,
		},
	}, nil
}

// handleRequestReview adds a review request to the session.
func (s *Server) handleRequestReview(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input RequestReviewInput,
) (*mcp.CallToolResult, RequestReviewOutput, error) {
	if input.RepoPath == "" {
		return nil, RequestReviewOutput{}, NewValidationError("repo_path", "cannot be empty")
	}
	if input.Summary == "" {
		return nil, RequestReviewOutput{}, NewValidationError("summary", "cannot be empty")
	}

	reviewCount, err := s.store.AddReviewRequest(s.session.TicketID, input.RepoPath, input.Summary)
	if err != nil {
		return nil, RequestReviewOutput{}, WrapTicketError(err)
	}

	// Move ticket to review status (idempotent - no-op if already in review or done)
	_, currentStatus, err := s.store.Get(s.session.TicketID)
	if err != nil {
		return nil, RequestReviewOutput{}, WrapTicketError(err)
	}
	if currentStatus != ticket.StatusReview && currentStatus != ticket.StatusDone {
		if err := s.store.Move(s.session.TicketID, ticket.StatusReview); err != nil {
			return nil, RequestReviewOutput{}, WrapTicketError(err)
		}
	}

	return nil, RequestReviewOutput{
		Success:     true,
		Message:     "Review request added. Wait for human approval.",
		ReviewCount: reviewCount,
	}, nil
}

// handleConcludeSession ends the session and moves the ticket to done.
func (s *Server) handleConcludeSession(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ConcludeSessionInput,
) (*mcp.CallToolResult, ConcludeSessionOutput, error) {
	if input.FullReport == "" {
		return nil, ConcludeSessionOutput{}, NewValidationError("full_report", "cannot be empty")
	}

	t, _, err := s.store.Get(s.session.TicketID)
	if err != nil {
		return nil, ConcludeSessionOutput{}, WrapTicketError(err)
	}

	// Capture worktree info before ending session
	var worktreePath, featureBranch *string
	var activeSessionID string
	if t.Session != nil {
		worktreePath = t.Session.WorktreePath
		featureBranch = t.Session.FeatureBranch
		if t.Session.IsActive() {
			activeSessionID = t.Session.ID
		}
	}

	// Add ticket_done comment with the full report
	_, err = s.store.AddComment(s.session.TicketID, activeSessionID, ticket.CommentTicketDone, input.FullReport)
	if err != nil {
		return nil, ConcludeSessionOutput{}, WrapTicketError(err)
	}

	// End the session
	if err := s.store.EndSession(s.session.TicketID); err != nil {
		return nil, ConcludeSessionOutput{}, WrapTicketError(err)
	}

	// Move the ticket to done
	if err := s.store.Move(s.session.TicketID, ticket.StatusDone); err != nil {
		return nil, ConcludeSessionOutput{}, WrapTicketError(err)
	}

	// Cleanup worktree if present
	if worktreePath != nil && featureBranch != nil && s.config.ProjectPath != "" {
		wm := worktree.NewManager(s.config.ProjectPath)
		if err := wm.Remove(*worktreePath, *featureBranch); err != nil {
			// Log but don't fail - main work is done
			log.Printf("warning: failed to cleanup worktree: %v", err)
		}
	}

	return nil, ConcludeSessionOutput{
		Success:  true,
		TicketID: s.session.TicketID,
		Message:  "Session concluded and ticket moved to done",
	}, nil
}
