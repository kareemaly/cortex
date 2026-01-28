package mcp

import (
	"context"

	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/types"
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

// handleReadOwnTicket reads the ticket assigned to this session via the daemon API.
func (s *Server) handleReadOwnTicket(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input EmptyInput,
) (*mcp.CallToolResult, ReadTicketOutput, error) {
	resp, err := s.sdkClient.GetTicketByID(s.session.TicketID)
	if err != nil {
		return nil, ReadTicketOutput{}, wrapSDKError(err)
	}

	return nil, ReadTicketOutput{
		Ticket: ticketResponseToOutput(resp),
	}, nil
}

// handleAddTicketComment adds a comment to the assigned ticket via the daemon API.
func (s *Server) handleAddTicketComment(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input AddCommentInput,
) (*mcp.CallToolResult, AddCommentOutput, error) {
	if input.Title == "" {
		return nil, AddCommentOutput{}, NewValidationError("title", "cannot be empty")
	}

	resp, err := s.sdkClient.AddComment(s.session.TicketID, input.Type, input.Title, input.Content)
	if err != nil {
		return nil, AddCommentOutput{}, wrapSDKError(err)
	}

	return nil, AddCommentOutput{
		Success: resp.Success,
		Comment: resp.Comment,
	}, nil
}

// handleRequestReview requests a review via the daemon API.
func (s *Server) handleRequestReview(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input RequestReviewInput,
) (*mcp.CallToolResult, RequestReviewOutput, error) {
	if input.RepoPath == "" {
		return nil, RequestReviewOutput{}, NewValidationError("repo_path", "cannot be empty")
	}
	if input.Title == "" {
		return nil, RequestReviewOutput{}, NewValidationError("title", "cannot be empty")
	}
	if input.Content == "" {
		return nil, RequestReviewOutput{}, NewValidationError("content", "cannot be empty")
	}

	resp, err := s.sdkClient.RequestReview(s.session.TicketID, input.RepoPath, input.Title, input.Content)
	if err != nil {
		return nil, RequestReviewOutput{}, wrapSDKError(err)
	}

	return nil, RequestReviewOutput{
		Success:     resp.Success,
		Message:     resp.Message,
		ReviewCount: resp.ReviewCount,
	}, nil
}

// handleConcludeSession concludes the session via the daemon API.
// All cleanup (worktree removal, tmux window kill) is handled by the daemon.
func (s *Server) handleConcludeSession(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ConcludeSessionInput,
) (*mcp.CallToolResult, ConcludeSessionOutput, error) {
	if input.FullReport == "" {
		return nil, ConcludeSessionOutput{}, NewValidationError("full_report", "cannot be empty")
	}

	resp, err := s.sdkClient.ConcludeSession(s.session.TicketID, input.FullReport)
	if err != nil {
		return nil, ConcludeSessionOutput{}, wrapSDKError(err)
	}

	return nil, ConcludeSessionOutput{
		Success:  resp.Success,
		TicketID: resp.TicketID,
		Message:  resp.Message,
	}, nil
}

// ticketResponseToOutput converts an SDK TicketResponse to an MCP TicketOutput.
func ticketResponseToOutput(r *types.TicketResponse) TicketOutput {
	var session *SessionOutput
	if r.Session != nil {
		s := sessionResponseToOutput(r.Session)
		session = &s
	}

	return TicketOutput{
		ID:       r.ID,
		Type:     r.Type,
		Title:    r.Title,
		Body:     r.Body,
		Status:   r.Status,
		Dates:    r.Dates,
		Comments: r.Comments,
		Session:  session,
	}
}

// sessionResponseToOutput converts an SDK SessionResponse to an MCP SessionOutput.
func sessionResponseToOutput(r *types.SessionResponse) SessionOutput {
	return SessionOutput{
		ID:            r.ID,
		StartedAt:     r.StartedAt,
		EndedAt:       r.EndedAt,
		Agent:         r.Agent,
		TmuxWindow:    r.TmuxWindow,
		CurrentStatus: r.CurrentStatus,
		IsActive:      r.EndedAt == nil,
	}
}

// wrapSDKError converts an SDK error to an MCP ToolError.
func wrapSDKError(err error) *ToolError {
	if err == nil {
		return nil
	}

	if apiErr, ok := err.(*sdk.APIError); ok {
		switch apiErr.Status {
		case 404:
			return &ToolError{Code: ErrorCodeNotFound, Message: apiErr.Message}
		case 400:
			return &ToolError{Code: ErrorCodeValidation, Message: apiErr.Message}
		case 409:
			return &ToolError{Code: ErrorCodeStateConflict, Message: apiErr.Message}
		default:
			return &ToolError{Code: ErrorCodeInternal, Message: apiErr.Message}
		}
	}

	return NewInternalError(err.Error())
}
