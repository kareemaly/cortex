package mcp

import (
	"context"
	"os"

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

	// Create follow-up ticket linked to the current ticket
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "createFollowUpTicket",
		Description: "Create a follow-up work ticket in backlog, automatically linked to the current ticket. The new ticket's references will include the current ticket ID, and the current ticket's references will be updated to include the new ticket ID.",
	}, s.handleCreateFollowUpTicket)

	// Conclude session tool
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "concludeSession",
		Description: "Conclude the session and mark the ticket as done. commits is required (at least one SHA). If no commits were produced, set rejected=true and provide a rejection_reason.",
	}, s.handleConcludeSession)
}

// handleConcludeSession concludes the session via the daemon API.
// All cleanup (tmux window kill) is handled by the daemon.
func (s *Server) handleConcludeSession(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ConcludeSessionInput,
) (*mcp.CallToolResult, ConcludeSessionOutput, error) {
	if input.Body == "" {
		return nil, ConcludeSessionOutput{}, NewValidationError("body", "cannot be empty")
	}

	startedAt := os.Getenv("CORTEX_STARTED_AT")
	resp, err := s.sdkClient.ConcludeSession(sdk.ConcludeSessionParams{
		TicketID:        s.session.TicketID,
		Body:            input.Body,
		StartedAt:       startedAt,
		Commits:         input.Commits,
		Rejected:        input.Rejected,
		RejectionReason: input.RejectionReason,
	})
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
		ID:            r.ID,
		Title:         r.Title,
		Body:          r.Body,
		Repo:          r.Repo,
		HasConclusion: r.HasConclusion,
		References:    r.References,
		Status:        r.Status,
		Created:       r.Created,
		Updated:       r.Updated,
		Due:           r.Due,
	}
}

func ticketResponseToMetadataOutput(r *types.TicketResponse) TicketMetadataOutput {
	return TicketMetadataOutput{
		ID:      r.ID,
		Title:   r.Title,
		Repo:    r.Repo,
		Status:  r.Status,
		Created: r.Created,
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
