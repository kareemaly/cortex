package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// handleCreateDoc creates a new doc.
func (s *Server) handleCreateDoc(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input CreateDocInput,
) (*mcp.CallToolResult, CreateDocOutput, error) {
	if err := s.validateProjectPath(input.ProjectPath); err != nil {
		return nil, CreateDocOutput{}, err
	}

	client := s.getClientForProject(input.ProjectPath)

	if input.Title == "" {
		return nil, CreateDocOutput{}, NewValidationError("title", "cannot be empty")
	}
	if input.Category == "" {
		return nil, CreateDocOutput{}, NewValidationError("category", "cannot be empty")
	}

	resp, err := client.CreateDoc(input.Title, input.Category, input.Body, input.Tags, input.References)
	if err != nil {
		return nil, CreateDocOutput{}, wrapSDKError(err)
	}

	return nil, CreateDocOutput{
		Doc: docResponseToOutput(resp),
	}, nil
}

// handleReadDoc reads a doc by ID.
func (s *Server) handleReadDoc(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ReadDocInput,
) (*mcp.CallToolResult, ReadDocOutput, error) {
	if err := s.validateProjectPath(input.ProjectPath); err != nil {
		return nil, ReadDocOutput{}, err
	}

	client := s.getClientForProject(input.ProjectPath)

	if input.ID == "" {
		return nil, ReadDocOutput{}, NewValidationError("id", "cannot be empty")
	}

	resp, err := client.GetDoc(input.ID)
	if err != nil {
		return nil, ReadDocOutput{}, wrapSDKError(err)
	}

	return nil, ReadDocOutput{
		Doc: docResponseToOutput(resp),
	}, nil
}

// handleUpdateDoc updates a doc.
func (s *Server) handleUpdateDoc(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input UpdateDocInput,
) (*mcp.CallToolResult, UpdateDocOutput, error) {
	if err := s.validateProjectPath(input.ProjectPath); err != nil {
		return nil, UpdateDocOutput{}, err
	}

	client := s.getClientForProject(input.ProjectPath)

	if input.ID == "" {
		return nil, UpdateDocOutput{}, NewValidationError("id", "cannot be empty")
	}

	resp, err := client.UpdateDoc(input.ID, input.Title, input.Body, input.Tags, input.References)
	if err != nil {
		return nil, UpdateDocOutput{}, wrapSDKError(err)
	}

	return nil, UpdateDocOutput{
		Doc: docResponseToOutput(resp),
	}, nil
}

// handleDeleteDoc deletes a doc (current project only).
func (s *Server) handleDeleteDoc(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input DeleteDocInput,
) (*mcp.CallToolResult, DeleteDocOutput, error) {
	if input.ID == "" {
		return nil, DeleteDocOutput{}, NewValidationError("id", "cannot be empty")
	}

	err := s.sdkClient.DeleteDoc(input.ID)
	if err != nil {
		return nil, DeleteDocOutput{}, wrapSDKError(err)
	}

	return nil, DeleteDocOutput{
		Success: true,
		ID:      input.ID,
	}, nil
}

// handleMoveDoc moves a doc to a different category.
func (s *Server) handleMoveDoc(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input MoveDocInput,
) (*mcp.CallToolResult, MoveDocOutput, error) {
	if err := s.validateProjectPath(input.ProjectPath); err != nil {
		return nil, MoveDocOutput{}, err
	}

	client := s.getClientForProject(input.ProjectPath)

	if input.ID == "" {
		return nil, MoveDocOutput{}, NewValidationError("id", "cannot be empty")
	}
	if input.Category == "" {
		return nil, MoveDocOutput{}, NewValidationError("category", "cannot be empty")
	}

	resp, err := client.MoveDoc(input.ID, input.Category)
	if err != nil {
		return nil, MoveDocOutput{}, wrapSDKError(err)
	}

	return nil, MoveDocOutput{
		Doc: docResponseToOutput(resp),
	}, nil
}

// handleListDocs lists docs with optional filters.
func (s *Server) handleListDocs(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ListDocsInput,
) (*mcp.CallToolResult, ListDocsOutput, error) {
	if err := s.validateProjectPath(input.ProjectPath); err != nil {
		return nil, ListDocsOutput{}, err
	}

	client := s.getClientForProject(input.ProjectPath)

	resp, err := client.ListDocs(input.Category, input.Tag, input.Query)
	if err != nil {
		return nil, ListDocsOutput{}, wrapSDKError(err)
	}

	summaries := make([]DocSummaryOutput, len(resp.Docs))
	for i, d := range resp.Docs {
		summaries[i] = docSummaryToOutput(&d)
	}

	return nil, ListDocsOutput{
		Docs:  summaries,
		Total: len(summaries),
	}, nil
}
