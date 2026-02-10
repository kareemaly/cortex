package mcp

import (
	"context"

	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/types"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerTicketTools registers all tools available to ticket sessions.
func (s *Server) registerTicketTools() {
	// Read reference (follow cross-references to tickets or docs)
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "readReference",
		Description: "Read a referenced ticket or doc by ID",
	}, s.handleReadReference)

	// Add comment tool
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "addComment",
		Description: "Add a comment to your assigned ticket",
	}, s.handleAddComment)

	// Add blocker tool
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "addBlocker",
		Description: "Report a blocker on your assigned ticket",
	}, s.handleAddBlocker)

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

	// createDoc — research tickets only
	if s.session.TicketType == "research" {
		mcp.AddTool(s.mcpServer, &mcp.Tool{
			Name:        "createDoc",
			Description: "Create a documentation file for research findings",
		}, s.handleTicketCreateDoc)
	}
}

// handleReadReference reads a referenced ticket or doc by ID via the daemon API.
func (s *Server) handleReadReference(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ReadReferenceInput,
) (*mcp.CallToolResult, ReadReferenceOutput, error) {
	if input.ID == "" {
		return nil, ReadReferenceOutput{}, NewValidationError("id", "cannot be empty")
	}
	if input.Type != "ticket" && input.Type != "doc" {
		return nil, ReadReferenceOutput{}, NewValidationError("type", "must be 'ticket' or 'doc'")
	}

	switch input.Type {
	case "ticket":
		resp, err := s.sdkClient.GetTicketByID(input.ID)
		if err != nil {
			return nil, ReadReferenceOutput{}, wrapSDKError(err)
		}
		out := ticketResponseToOutput(resp)
		return nil, ReadReferenceOutput{Ticket: &out}, nil

	case "doc":
		resp, err := s.sdkClient.GetDoc(input.ID)
		if err != nil {
			return nil, ReadReferenceOutput{}, wrapSDKError(err)
		}
		out := docResponseToOutput(resp)
		return nil, ReadReferenceOutput{Doc: &out}, nil

	default:
		return nil, ReadReferenceOutput{}, NewValidationError("type", "must be 'ticket' or 'doc'")
	}
}

// handleTicketCreateDoc creates a doc for research ticket agents via the daemon API.
func (s *Server) handleTicketCreateDoc(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input TicketCreateDocInput,
) (*mcp.CallToolResult, CreateDocOutput, error) {
	if input.Title == "" {
		return nil, CreateDocOutput{}, NewValidationError("title", "cannot be empty")
	}
	if input.Category == "" {
		return nil, CreateDocOutput{}, NewValidationError("category", "cannot be empty")
	}

	resp, err := s.sdkClient.CreateDoc(input.Title, input.Category, input.Body, input.Tags, nil)
	if err != nil {
		return nil, CreateDocOutput{}, wrapSDKError(err)
	}

	return nil, CreateDocOutput{
		Doc: docResponseToOutput(resp),
	}, nil
}

// handleAddComment adds a general comment to the assigned ticket via the daemon API.
func (s *Server) handleAddComment(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input AddCommentInput,
) (*mcp.CallToolResult, AddCommentOutput, error) {
	if input.Content == "" {
		return nil, AddCommentOutput{}, NewValidationError("content", "cannot be empty")
	}

	resp, err := s.sdkClient.AddComment(s.session.TicketID, "comment", input.Content, "")
	if err != nil {
		return nil, AddCommentOutput{}, wrapSDKError(err)
	}

	return nil, AddCommentOutput{
		Success: resp.Success,
		Comment: resp.Comment,
	}, nil
}

// handleAddBlocker adds a blocker comment to the assigned ticket via the daemon API.
func (s *Server) handleAddBlocker(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input AddBlockerInput,
) (*mcp.CallToolResult, AddCommentOutput, error) {
	if input.Content == "" {
		return nil, AddCommentOutput{}, NewValidationError("content", "cannot be empty")
	}

	resp, err := s.sdkClient.AddComment(s.session.TicketID, "blocker", input.Content, "")
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
	if input.Content == "" {
		return nil, RequestReviewOutput{}, NewValidationError("content", "cannot be empty")
	}

	resp, err := s.sdkClient.RequestReview(s.session.TicketID, input.RepoPath, input.Content, input.Commit)
	if err != nil {
		return nil, RequestReviewOutput{}, wrapSDKError(err)
	}

	return nil, RequestReviewOutput{
		Success: resp.Success,
		Message: resp.Message,
		Comment: resp.Comment,
	}, nil
}

// handleConcludeSession concludes the session via the daemon API.
// All cleanup (worktree removal, tmux window kill) is handled by the daemon.
func (s *Server) handleConcludeSession(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ConcludeSessionInput,
) (*mcp.CallToolResult, ConcludeSessionOutput, error) {
	if input.Content == "" {
		return nil, ConcludeSessionOutput{}, NewValidationError("content", "cannot be empty")
	}

	resp, err := s.sdkClient.ConcludeSession(s.session.TicketID, input.Content)
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
	return TicketOutput{
		ID:         r.ID,
		Type:       r.Type,
		Title:      r.Title,
		Body:       r.Body,
		Tags:       r.Tags,
		References: r.References,
		Status:     r.Status,
		Created:    r.Created,
		Updated:    r.Updated,
		Due:        r.Due,
		Comments:   r.Comments,
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
