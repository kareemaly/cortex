package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// handleListNotes lists all notes for a project.
func (s *Server) handleListNotes(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ListNotesInput,
) (*mcp.CallToolResult, ListNotesOutput, error) {
	if err := s.validateProjectPath(input.ProjectPath); err != nil {
		return nil, ListNotesOutput{}, err
	}

	client := s.getClientForProject(input.ProjectPath)

	resp, err := client.ListNotes()
	if err != nil {
		return nil, ListNotesOutput{}, wrapSDKError(err)
	}

	notes := make([]NoteOutput, len(resp.Notes))
	for i, n := range resp.Notes {
		notes[i] = noteResponseToOutput(&n)
	}

	return nil, ListNotesOutput{
		Notes: notes,
		Total: len(notes),
	}, nil
}

// handleCreateNote creates a new note.
func (s *Server) handleCreateNote(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input CreateNoteInput,
) (*mcp.CallToolResult, CreateNoteOutput, error) {
	if err := s.validateProjectPath(input.ProjectPath); err != nil {
		return nil, CreateNoteOutput{}, err
	}

	client := s.getClientForProject(input.ProjectPath)

	if input.Text == "" {
		return nil, CreateNoteOutput{}, NewValidationError("text", "cannot be empty")
	}

	var due *string
	if input.Due != "" {
		due = &input.Due
	}

	resp, err := client.CreateNote(input.Text, due)
	if err != nil {
		return nil, CreateNoteOutput{}, wrapSDKError(err)
	}

	return nil, CreateNoteOutput{
		Note: noteResponseToOutput(resp),
	}, nil
}

// handleUpdateNote updates a note's text and/or due date.
func (s *Server) handleUpdateNote(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input UpdateNoteInput,
) (*mcp.CallToolResult, UpdateNoteOutput, error) {
	if err := s.validateProjectPath(input.ProjectPath); err != nil {
		return nil, UpdateNoteOutput{}, err
	}

	client := s.getClientForProject(input.ProjectPath)

	if input.ID == "" {
		return nil, UpdateNoteOutput{}, NewValidationError("id", "cannot be empty")
	}

	resp, err := client.UpdateNote(input.ID, input.Text, input.Due)
	if err != nil {
		return nil, UpdateNoteOutput{}, wrapSDKError(err)
	}

	return nil, UpdateNoteOutput{
		Note: noteResponseToOutput(resp),
	}, nil
}

// handleDeleteNote deletes a note by ID (current project only).
func (s *Server) handleDeleteNote(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input DeleteNoteInput,
) (*mcp.CallToolResult, DeleteNoteOutput, error) {
	if input.ID == "" {
		return nil, DeleteNoteOutput{}, NewValidationError("id", "cannot be empty")
	}

	err := s.sdkClient.DeleteNote(input.ID)
	if err != nil {
		return nil, DeleteNoteOutput{}, wrapSDKError(err)
	}

	return nil, DeleteNoteOutput{
		Success: true,
		ID:      input.ID,
	}, nil
}
