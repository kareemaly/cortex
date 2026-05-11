package mcp

import (
	"context"
	"os"

	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerCollabTools registers all tools available to collab sessions.
func (s *Server) registerCollabTools() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "createWorkTicket",
		Description: "Create a new work ticket in backlog. Requires a repo field — provide a stable repo key from cortex.yaml.",
	}, s.handleCreateWorkTicket)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "updateTicket",
		Description: "Update mutable ticket fields. Accepts: id (required), title, body, dueDate, references. dueDate must be RFC3339 when set, and an explicit empty string clears it. Use editTicketBody for targeted body edits; keep updateTicket for full-body rewrites. Does NOT support updating type, repo, status, or any other fields.",
	}, s.handleUpdateTicket)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "editTicketBody",
		Description: "Edit part of a ticket body using oldString/newString replacement. Preferred over updateTicket for body edits because it avoids full-body JSON serialization issues.",
	}, s.handleEditTicketBody)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "createFollowUpTicket",
		Description: "Create a follow-up work ticket in backlog. Use this when you identify work that should be tracked separately.",
	}, s.handleCreateFollowUpTicket)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "concludeSession",
		Description: "Conclude the collab session and create a conclusion record. Include the outcome, files changed, and follow-up work or blockers. commits is optional.",
	}, s.handleCollabConcludeSession)
}

// handleCollabConcludeSession concludes the collab session via the daemon API.
func (s *Server) handleCollabConcludeSession(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ConcludeSessionInput,
) (*mcp.CallToolResult, CollabConcludeOutput, error) {
	if input.Body == "" {
		return nil, CollabConcludeOutput{}, NewValidationError("body", "cannot be empty")
	}

	collabID := s.session.CollabID
	startedAt := os.Getenv("CORTEX_STARTED_AT")

	resp, err := s.sdkClient.ConcludeCollabSession(sdk.ConcludeCollabSessionParams{
		CollabID:  collabID,
		Body:      input.Body,
		StartedAt: startedAt,
		Commits:   input.Commits,
	})
	if err != nil {
		return nil, CollabConcludeOutput{}, wrapSDKError(err)
	}

	return nil, CollabConcludeOutput{
		Success:  resp.Success,
		CollabID: collabID,
		Message:  resp.Message,
	}, nil
}
