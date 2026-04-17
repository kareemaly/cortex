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
		Description: "Create a new work ticket in backlog. Requires a repo field — the agent will spawn in that repo directory.",
	}, s.handleCreateWorkTicket)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "updateTicket",
		Description: "Update ticket fields. Only accepts: id (required), title, body, references. Does NOT support updating type, repo, path, status, due_date, or any other fields.",
	}, s.handleUpdateTicket)

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
