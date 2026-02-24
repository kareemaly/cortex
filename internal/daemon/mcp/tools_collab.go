package mcp

import (
	"context"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerCollabTools registers all tools available to collab sessions.
func (s *Server) registerCollabTools() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "concludeSession",
		Description: "Conclude the collab session and create a conclusion record.",
	}, s.handleCollabConcludeSession)
}

// handleCollabConcludeSession concludes the collab session via the daemon API.
func (s *Server) handleCollabConcludeSession(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ConcludeSessionInput,
) (*mcp.CallToolResult, CollabConcludeOutput, error) {
	if input.Content == "" {
		return nil, CollabConcludeOutput{}, NewValidationError("content", "cannot be empty")
	}

	collabID := s.session.CollabID
	startedAt := os.Getenv("CORTEX_STARTED_AT")

	resp, err := s.sdkClient.ConcludeCollabSession(collabID, input.Content, startedAt)
	if err != nil {
		return nil, CollabConcludeOutput{}, wrapSDKError(err)
	}

	return nil, CollabConcludeOutput{
		Success:  resp.Success,
		CollabID: collabID,
		Message:  resp.Message,
	}, nil
}
