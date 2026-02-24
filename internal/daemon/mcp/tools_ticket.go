package mcp

import (
	"context"

	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/types"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerTicketTools registers all tools available to ticket sessions.
func (s *Server) registerTicketTools() {
	// Read ticket (for cross-reference context)
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "readTicket",
		Description: "Read full ticket details by ID. Use this to get context on referenced tickets.",
	}, s.handleReadTicket)

	// Conclude session tool
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "concludeSession",
		Description: "Conclude the session and mark the ticket as done. Call this after all reviews are approved.",
	}, s.handleConcludeSession)
}

// handleConcludeSession concludes the session via the daemon API.
// All cleanup (tmux window kill) is handled by the daemon.
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
		Repo:       r.Repo,
		Path:       r.Path,
		Session:    r.Session,
		References: r.References,
		Status:     r.Status,
		Created:    r.Created,
		Updated:    r.Updated,
		Due:        r.Due,
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
