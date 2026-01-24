package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kareemaly/cortex/internal/prompt"
	"github.com/kareemaly/cortex/internal/ticket"
	"github.com/kareemaly/cortex/internal/tmux"
)

// setupTestServerWithMockTmux creates a test server with a mock tmux manager.
func setupTestServerWithMockTmux(t *testing.T, ticketID string) (*Server, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "mcp-server-test")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	// Create the prompts directory and default templates for the project
	promptsDir := filepath.Join(tmpDir, ".cortex", "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("create prompts dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(promptsDir, "ticket-agent.md"), []byte(prompt.DefaultTicketAgentPrompt), 0644); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("create ticket-agent.md: %v", err)
	}

	mockRunner := tmux.NewMockRunner()
	tmuxMgr := tmux.NewManagerWithRunner(mockRunner)

	cfg := &Config{
		TicketID:    ticketID,
		TicketsDir:  tmpDir,
		ProjectPath: tmpDir,
		TmuxSession: "test-session",
		TmuxManager: tmuxMgr,
		CortexdPath: "/mock/cortexd",
	}

	server, err := NewServer(cfg)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("create server: %v", err)
	}

	cleanup := func() {
		_ = os.RemoveAll(tmpDir)
	}

	return server, cleanup
}

// Architect tool tests

func TestHandleListTickets(t *testing.T) {
	server, cleanup := setupTestServer(t, "")
	defer cleanup()

	// Create some tickets
	_, _ = server.Store().Create("Ticket 1", "body 1")
	_, _ = server.Store().Create("Ticket 2", "body 2")

	// List backlog tickets (status is required)
	_, output, err := server.handleListTickets(context.Background(), nil, ListTicketsInput{
		Status: "backlog",
	})
	if err != nil {
		t.Fatalf("handleListTickets failed: %v", err)
	}

	if output.Total != 2 {
		t.Errorf("total = %d, want 2", output.Total)
	}
	if len(output.Tickets) != 2 {
		t.Errorf("len(tickets) = %d, want 2", len(output.Tickets))
	}
}

func TestHandleListTicketsWithStatus(t *testing.T) {
	server, cleanup := setupTestServer(t, "")
	defer cleanup()

	// Create tickets in different statuses
	t1, _ := server.Store().Create("Backlog Ticket", "")
	t2, _ := server.Store().Create("Progress Ticket", "")
	_ = server.Store().Move(t2.ID, ticket.StatusProgress)

	// List only backlog
	_, output, err := server.handleListTickets(context.Background(), nil, ListTicketsInput{
		Status: "backlog",
	})
	if err != nil {
		t.Fatalf("handleListTickets failed: %v", err)
	}

	if output.Total != 1 {
		t.Errorf("total = %d, want 1", output.Total)
	}
	if output.Tickets[0].ID != t1.ID {
		t.Error("wrong ticket returned")
	}
}

func TestHandleListTicketsWithQuery(t *testing.T) {
	server, cleanup := setupTestServer(t, "")
	defer cleanup()

	// Create tickets
	_, _ = server.Store().Create("Fix login bug", "Authentication issue")
	_, _ = server.Store().Create("Add feature", "New feature")

	// Search for "login" in backlog (status is required)
	_, output, err := server.handleListTickets(context.Background(), nil, ListTicketsInput{
		Status: "backlog",
		Query:  "login",
	})
	if err != nil {
		t.Fatalf("handleListTickets failed: %v", err)
	}

	if output.Total != 1 {
		t.Errorf("total = %d, want 1", output.Total)
	}
}

func TestHandleListTicketsWithStatusAndQuery(t *testing.T) {
	server, cleanup := setupTestServer(t, "")
	defer cleanup()

	// Create tickets
	t1, _ := server.Store().Create("Fix login bug", "Authentication issue")
	t2, _ := server.Store().Create("Fix login feature", "Another login issue")
	_ = server.Store().Move(t2.ID, ticket.StatusProgress)

	// Search for "login" in backlog only
	_, output, err := server.handleListTickets(context.Background(), nil, ListTicketsInput{
		Status: "backlog",
		Query:  "login",
	})
	if err != nil {
		t.Fatalf("handleListTickets failed: %v", err)
	}

	if output.Total != 1 {
		t.Errorf("total = %d, want 1", output.Total)
	}
	if output.Tickets[0].ID != t1.ID {
		t.Errorf("wrong ticket returned, expected %s got %s", t1.ID, output.Tickets[0].ID)
	}
}

func TestHandleListTicketsEmptyQuery(t *testing.T) {
	server, cleanup := setupTestServer(t, "")
	defer cleanup()

	// Create tickets
	_, _ = server.Store().Create("Ticket 1", "body 1")
	_, _ = server.Store().Create("Ticket 2", "body 2")

	// Empty query should return all tickets in the specified status (status is required)
	_, output, err := server.handleListTickets(context.Background(), nil, ListTicketsInput{
		Status: "backlog",
		Query:  "",
	})
	if err != nil {
		t.Fatalf("handleListTickets failed: %v", err)
	}

	if output.Total != 2 {
		t.Errorf("total = %d, want 2", output.Total)
	}
}

func TestHandleListTickets_MissingStatus(t *testing.T) {
	server, cleanup := setupTestServer(t, "")
	defer cleanup()

	// Call without status - should return validation error
	_, _, err := server.handleListTickets(context.Background(), nil, ListTicketsInput{})

	if err == nil {
		t.Error("expected error for missing status")
	}
	toolErr, ok := err.(*ToolError)
	if !ok {
		t.Fatalf("expected ToolError, got %T", err)
	}
	if toolErr.Code != ErrorCodeValidation {
		t.Errorf("error code = %q, want %q", toolErr.Code, ErrorCodeValidation)
	}
}

func TestHandleListTickets_InvalidStatus(t *testing.T) {
	server, cleanup := setupTestServer(t, "")
	defer cleanup()

	// Call with invalid status - should return validation error
	_, _, err := server.handleListTickets(context.Background(), nil, ListTicketsInput{
		Status: "invalid",
	})

	if err == nil {
		t.Error("expected error for invalid status")
	}
	toolErr, ok := err.(*ToolError)
	if !ok {
		t.Fatalf("expected ToolError, got %T", err)
	}
	if toolErr.Code != ErrorCodeValidation {
		t.Errorf("error code = %q, want %q", toolErr.Code, ErrorCodeValidation)
	}
}

func TestHandleReadTicket(t *testing.T) {
	server, cleanup := setupTestServer(t, "")
	defer cleanup()

	created, _ := server.Store().Create("Test Ticket", "body")

	_, output, err := server.handleReadTicket(context.Background(), nil, ReadTicketInput{
		ID: created.ID,
	})
	if err != nil {
		t.Fatalf("handleReadTicket failed: %v", err)
	}

	if output.Ticket.ID != created.ID {
		t.Errorf("ID = %q, want %q", output.Ticket.ID, created.ID)
	}
	if output.Ticket.Title != "Test Ticket" {
		t.Errorf("title = %q, want %q", output.Ticket.Title, "Test Ticket")
	}
}

func TestHandleReadTicketNotFound(t *testing.T) {
	server, cleanup := setupTestServer(t, "")
	defer cleanup()

	_, _, err := server.handleReadTicket(context.Background(), nil, ReadTicketInput{
		ID: "nonexistent",
	})

	if err == nil {
		t.Error("expected error for nonexistent ticket")
	}
	toolErr, ok := err.(*ToolError)
	if !ok {
		t.Fatalf("expected ToolError, got %T", err)
	}
	if toolErr.Code != ErrorCodeNotFound {
		t.Errorf("error code = %q, want %q", toolErr.Code, ErrorCodeNotFound)
	}
}

func TestHandleCreateTicket(t *testing.T) {
	server, cleanup := setupTestServer(t, "")
	defer cleanup()

	_, output, err := server.handleCreateTicket(context.Background(), nil, CreateTicketInput{
		Title: "New Ticket",
		Body:  "Description",
	})
	if err != nil {
		t.Fatalf("handleCreateTicket failed: %v", err)
	}

	if output.Ticket.ID == "" {
		t.Error("ticket ID should not be empty")
	}
	if output.Ticket.Title != "New Ticket" {
		t.Errorf("title = %q, want %q", output.Ticket.Title, "New Ticket")
	}
	if output.Ticket.Status != "backlog" {
		t.Errorf("status = %q, want %q", output.Ticket.Status, "backlog")
	}
}

func TestHandleCreateTicketEmptyTitle(t *testing.T) {
	server, cleanup := setupTestServer(t, "")
	defer cleanup()

	_, _, err := server.handleCreateTicket(context.Background(), nil, CreateTicketInput{
		Title: "",
	})

	if err == nil {
		t.Error("expected error for empty title")
	}
}

func TestHandleUpdateTicket(t *testing.T) {
	server, cleanup := setupTestServer(t, "")
	defer cleanup()

	created, _ := server.Store().Create("Original", "body")
	newTitle := "Updated"

	_, output, err := server.handleUpdateTicket(context.Background(), nil, UpdateTicketInput{
		ID:    created.ID,
		Title: &newTitle,
	})
	if err != nil {
		t.Fatalf("handleUpdateTicket failed: %v", err)
	}

	if output.Ticket.Title != "Updated" {
		t.Errorf("title = %q, want %q", output.Ticket.Title, "Updated")
	}
}

func TestHandleDeleteTicket(t *testing.T) {
	server, cleanup := setupTestServer(t, "")
	defer cleanup()

	created, _ := server.Store().Create("To Delete", "")

	_, output, err := server.handleDeleteTicket(context.Background(), nil, DeleteTicketInput{
		ID: created.ID,
	})
	if err != nil {
		t.Fatalf("handleDeleteTicket failed: %v", err)
	}

	if !output.Success {
		t.Error("delete should succeed")
	}

	// Verify deleted
	_, _, err = server.Store().Get(created.ID)
	if !ticket.IsNotFound(err) {
		t.Error("ticket should be deleted")
	}
}

func TestHandleMoveTicket(t *testing.T) {
	server, cleanup := setupTestServer(t, "")
	defer cleanup()

	created, _ := server.Store().Create("Test", "")

	_, output, err := server.handleMoveTicket(context.Background(), nil, MoveTicketInput{
		ID:     created.ID,
		Status: "progress",
	})
	if err != nil {
		t.Fatalf("handleMoveTicket failed: %v", err)
	}

	if !output.Success {
		t.Error("move should succeed")
	}
	if output.Status != "progress" {
		t.Errorf("status = %q, want %q", output.Status, "progress")
	}
}

func TestHandleMoveTicketInvalidStatus(t *testing.T) {
	server, cleanup := setupTestServer(t, "")
	defer cleanup()

	created, _ := server.Store().Create("Test", "")

	_, _, err := server.handleMoveTicket(context.Background(), nil, MoveTicketInput{
		ID:     created.ID,
		Status: "invalid",
	})

	if err == nil {
		t.Error("expected error for invalid status")
	}
}

func TestHandleSpawnSession(t *testing.T) {
	server, cleanup := setupTestServerWithMockTmux(t, "")
	defer cleanup()

	// Create a ticket first
	created, err := server.Store().Create("Test Spawn Session", "Test body")
	if err != nil {
		t.Fatalf("failed to create ticket: %v", err)
	}

	_, output, err := server.handleSpawnSession(context.Background(), nil, SpawnSessionInput{
		TicketID: created.ID,
	})
	if err != nil {
		t.Fatalf("handleSpawnSession failed: %v", err)
	}

	// With mock tmux, it should always succeed
	if !output.Success {
		t.Errorf("expected success, got message: %s", output.Message)
	}

	// Should return the ticket ID
	if output.TicketID != created.ID {
		t.Errorf("expected ticket_id %s, got %s", created.ID, output.TicketID)
	}

	// Should have a session ID
	if output.SessionID == "" {
		t.Error("expected session ID to be set")
	}

	// Should have a tmux window name
	if output.TmuxWindow == "" {
		t.Error("expected tmux window name")
	}
}

func TestHandleSpawnSessionEmptyTicketID(t *testing.T) {
	server, cleanup := setupTestServer(t, "")
	defer cleanup()

	_, _, err := server.handleSpawnSession(context.Background(), nil, SpawnSessionInput{
		TicketID: "",
	})
	if err == nil {
		t.Error("expected error for empty ticket_id")
	}
}

func TestHandleSpawnSessionTicketNotFound(t *testing.T) {
	server, cleanup := setupTestServer(t, "")
	defer cleanup()

	_, _, err := server.handleSpawnSession(context.Background(), nil, SpawnSessionInput{
		TicketID: "nonexistent-ticket",
	})
	if err == nil {
		t.Error("expected error for nonexistent ticket")
	}
}

func TestHandleSpawnSessionActiveSession(t *testing.T) {
	server, cleanup := setupTestServer(t, "")
	defer cleanup()

	// Create a ticket with an active session
	created, err := server.Store().Create("Test Active Session", "Test body")
	if err != nil {
		t.Fatalf("failed to create ticket: %v", err)
	}
	_, err = server.Store().SetSession(created.ID, "claude", "window")
	if err != nil {
		t.Fatalf("failed to set session: %v", err)
	}

	_, output, err := server.handleSpawnSession(context.Background(), nil, SpawnSessionInput{
		TicketID: created.ID,
	})
	if err != nil {
		t.Fatalf("handleSpawnSession failed: %v", err)
	}

	// Should fail because ticket already has active session
	if output.Success {
		t.Error("expected failure for ticket with active session")
	}
	if output.Message == "" {
		t.Error("expected error message")
	}
}

func TestHandleSpawnSessionNoAutoMove(t *testing.T) {
	server, cleanup := setupTestServerWithMockTmux(t, "")
	defer cleanup()

	// Create a ticket in backlog
	created, err := server.Store().Create("Test No Auto Move", "Test body")
	if err != nil {
		t.Fatalf("failed to create ticket: %v", err)
	}

	// Verify ticket is in backlog before spawn
	_, statusBefore, err := server.Store().Get(created.ID)
	if err != nil {
		t.Fatalf("failed to get ticket: %v", err)
	}
	if statusBefore != ticket.StatusBacklog {
		t.Fatalf("ticket should be in backlog before spawn, got %v", statusBefore)
	}

	// Spawn a session
	_, output, err := server.handleSpawnSession(context.Background(), nil, SpawnSessionInput{
		TicketID: created.ID,
	})
	if err != nil {
		t.Fatalf("handleSpawnSession failed: %v", err)
	}
	if !output.Success {
		t.Fatalf("spawn should succeed, got message: %s", output.Message)
	}

	// Verify ticket stays in backlog (no auto-move)
	_, statusAfter, err := server.Store().Get(created.ID)
	if err != nil {
		t.Fatalf("failed to get ticket after spawn: %v", err)
	}
	if statusAfter != ticket.StatusBacklog {
		t.Errorf("ticket should remain in backlog after spawn, got %v", statusAfter)
	}
}

// Ticket tool tests

func setupTicketSession(t *testing.T) (*Server, string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "mcp-ticket-test")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	// First create a store to make a ticket
	store, err := ticket.NewStore(tmpDir)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("create store: %v", err)
	}

	// Create a ticket
	tk, err := store.Create("Test Ticket", "body")
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("create ticket: %v", err)
	}

	// Set a session on the ticket
	_, err = store.SetSession(tk.ID, "claude", "window")
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("set session: %v", err)
	}

	// Now create server with ticket session
	cfg := &Config{
		TicketID:   tk.ID,
		TicketsDir: tmpDir,
	}

	server, err := NewServer(cfg)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("create server: %v", err)
	}

	cleanup := func() {
		_ = os.RemoveAll(tmpDir)
	}

	return server, tk.ID, cleanup
}

func TestHandleReadOwnTicket(t *testing.T) {
	server, ticketID, cleanup := setupTicketSession(t)
	defer cleanup()

	_, output, err := server.handleReadOwnTicket(context.Background(), nil, EmptyInput{})
	if err != nil {
		t.Fatalf("handleReadOwnTicket failed: %v", err)
	}

	if output.Ticket.ID != ticketID {
		t.Errorf("ID = %q, want %q", output.Ticket.ID, ticketID)
	}
}

// New tool tests

func TestHandleAddTicketComment(t *testing.T) {
	server, _, cleanup := setupTicketSession(t)
	defer cleanup()

	_, output, err := server.handleAddTicketComment(context.Background(), nil, AddCommentInput{
		Type:    "decision",
		Content: "Decided to use new API",
	})
	if err != nil {
		t.Fatalf("handleAddTicketComment failed: %v", err)
	}

	if !output.Success {
		t.Error("add comment should succeed")
	}
	if output.Comment.ID == "" {
		t.Error("comment ID should not be empty")
	}
	if output.Comment.Type != "decision" {
		t.Errorf("comment type = %q, want 'decision'", output.Comment.Type)
	}
	if output.Comment.Content != "Decided to use new API" {
		t.Errorf("comment content = %q, want 'Decided to use new API'", output.Comment.Content)
	}

	// Verify comment was added
	tk, _, _ := server.Store().Get(server.Session().TicketID)
	if len(tk.Comments) != 1 {
		t.Errorf("comments count = %d, want 1", len(tk.Comments))
	}
}

func TestHandleAddTicketCommentInvalidType(t *testing.T) {
	server, _, cleanup := setupTicketSession(t)
	defer cleanup()

	_, _, err := server.handleAddTicketComment(context.Background(), nil, AddCommentInput{
		Type:    "invalid_type",
		Content: "Test content",
	})
	if err == nil {
		t.Error("expected error for invalid comment type")
	}
}

func TestHandleMoveTicketToReview(t *testing.T) {
	server, _, cleanup := setupTicketSession(t)
	defer cleanup()

	// First move to progress
	_ = server.Store().Move(server.Session().TicketID, ticket.StatusProgress)

	_, output, err := server.handleMoveTicketToReview(context.Background(), nil, EmptyInput{})
	if err != nil {
		t.Fatalf("handleMoveTicketToReview failed: %v", err)
	}

	if !output.Success {
		t.Error("move to review should succeed")
	}
	if output.Message != "Ticket moved to review" {
		t.Errorf("message = %q, want 'Ticket moved to review'", output.Message)
	}

	// Verify status
	_, status, _ := server.Store().Get(server.Session().TicketID)
	if status != ticket.StatusReview {
		t.Errorf("status = %q, want %q", status, ticket.StatusReview)
	}
}

func TestHandleMoveTicketToReviewAlreadyInReview(t *testing.T) {
	server, _, cleanup := setupTicketSession(t)
	defer cleanup()

	_ = server.Store().Move(server.Session().TicketID, ticket.StatusReview)

	_, output, err := server.handleMoveTicketToReview(context.Background(), nil, EmptyInput{})
	if err != nil {
		t.Fatalf("handleMoveTicketToReview failed: %v", err)
	}

	if !output.Success {
		t.Error("should still succeed")
	}
	if output.Message != "Ticket is already in review" {
		t.Errorf("message = %q, want 'Ticket is already in review'", output.Message)
	}
}

func TestHandleMoveTicketToDone(t *testing.T) {
	server, _, cleanup := setupTicketSession(t)
	defer cleanup()

	_, output, err := server.handleMoveTicketToDone(context.Background(), nil, MoveTicketToDoneInput{})
	if err != nil {
		t.Fatalf("handleMoveTicketToDone failed: %v", err)
	}

	if !output.Success {
		t.Error("move to done should succeed")
	}
	if output.Status != "done" {
		t.Errorf("status = %q, want 'done'", output.Status)
	}

	// Verify status
	_, status, _ := server.Store().Get(server.Session().TicketID)
	if status != ticket.StatusDone {
		t.Errorf("status = %q, want %q", status, ticket.StatusDone)
	}
}

func TestHandleMoveTicketToDone_WithSummary(t *testing.T) {
	server, _, cleanup := setupTicketSession(t)
	defer cleanup()

	_, output, err := server.handleMoveTicketToDone(context.Background(), nil, MoveTicketToDoneInput{
		Summary: "Completed all tasks successfully",
	})
	if err != nil {
		t.Fatalf("handleMoveTicketToDone failed: %v", err)
	}

	if !output.Success {
		t.Error("move to done should succeed")
	}
	if output.Status != "done" {
		t.Errorf("status = %q, want 'done'", output.Status)
	}

	// Verify status
	tk, status, _ := server.Store().Get(server.Session().TicketID)
	if status != ticket.StatusDone {
		t.Errorf("status = %q, want %q", status, ticket.StatusDone)
	}

	// Verify summary was added as ticket_done comment
	if len(tk.Comments) != 1 {
		t.Errorf("comments count = %d, want 1", len(tk.Comments))
	}
	if len(tk.Comments) > 0 {
		if tk.Comments[0].Type != ticket.CommentTicketDone {
			t.Errorf("comment type = %q, want %q", tk.Comments[0].Type, ticket.CommentTicketDone)
		}
		if tk.Comments[0].Content != "Completed all tasks successfully" {
			t.Errorf("comment content = %q, want 'Completed all tasks successfully'", tk.Comments[0].Content)
		}
	}
}

// Error handling tests

func TestWrapTicketError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode ErrorCode
	}{
		{
			name:     "nil error",
			err:      nil,
			wantCode: "",
		},
		{
			name:     "NotFoundError",
			err:      &ticket.NotFoundError{Resource: "ticket", ID: "123"},
			wantCode: ErrorCodeNotFound,
		},
		{
			name:     "ValidationError",
			err:      &ticket.ValidationError{Field: "title", Message: "empty"},
			wantCode: ErrorCodeValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapTicketError(tt.err)
			if tt.err == nil {
				if result != nil {
					t.Error("expected nil result for nil error")
				}
				return
			}
			if result.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", result.Code, tt.wantCode)
			}
		})
	}
}

// SpawnSession state/mode matrix tests

func TestHandleSpawnSession_StateNormal_ModeNormal(t *testing.T) {
	server, cleanup := setupTestServerWithMockTmux(t, "")
	defer cleanup()

	created, _ := server.Store().Create("Test Ticket", "body")

	_, output, err := server.handleSpawnSession(context.Background(), nil, SpawnSessionInput{
		TicketID: created.ID,
		Mode:     "normal",
	})
	if err != nil {
		t.Fatalf("handleSpawnSession failed: %v", err)
	}

	if !output.Success {
		t.Errorf("expected success, got message: %s", output.Message)
	}
	if output.State != "normal" {
		t.Errorf("expected state 'normal', got: %s", output.State)
	}
}

func TestHandleSpawnSession_StateNormal_ModeResume(t *testing.T) {
	server, cleanup := setupTestServerWithMockTmux(t, "")
	defer cleanup()

	created, _ := server.Store().Create("Test Ticket", "body")

	_, output, err := server.handleSpawnSession(context.Background(), nil, SpawnSessionInput{
		TicketID: created.ID,
		Mode:     "resume",
	})

	if err == nil {
		t.Fatal("expected STATE_CONFLICT error")
	}
	toolErr, ok := err.(*ToolError)
	if !ok {
		t.Fatalf("expected ToolError, got %T", err)
	}
	if toolErr.Code != ErrorCodeStateConflict {
		t.Errorf("expected STATE_CONFLICT, got: %s", toolErr.Code)
	}
	if output.State != "normal" {
		t.Errorf("expected state 'normal', got: %s", output.State)
	}
}

func TestHandleSpawnSession_StateNormal_ModeFresh(t *testing.T) {
	server, cleanup := setupTestServerWithMockTmux(t, "")
	defer cleanup()

	created, _ := server.Store().Create("Test Ticket", "body")

	_, output, err := server.handleSpawnSession(context.Background(), nil, SpawnSessionInput{
		TicketID: created.ID,
		Mode:     "fresh",
	})

	if err == nil {
		t.Fatal("expected STATE_CONFLICT error")
	}
	toolErr, ok := err.(*ToolError)
	if !ok {
		t.Fatalf("expected ToolError, got %T", err)
	}
	if toolErr.Code != ErrorCodeStateConflict {
		t.Errorf("expected STATE_CONFLICT, got: %s", toolErr.Code)
	}
	if output.State != "normal" {
		t.Errorf("expected state 'normal', got: %s", output.State)
	}
}

func TestHandleSpawnSession_StateActive_AllModes(t *testing.T) {
	modes := []string{"normal", "resume", "fresh"}

	for _, mode := range modes {
		t.Run("mode="+mode, func(t *testing.T) {
			server, cleanup := setupTestServerWithMockTmux(t, "")
			defer cleanup()

			// Create ticket with active session (window exists because mock defaults to true for existing sessions)
			created, _ := server.Store().Create("Test Ticket", "body")
			_, _ = server.Store().SetSession(created.ID, "claude", "window")

			_, output, err := server.handleSpawnSession(context.Background(), nil, SpawnSessionInput{
				TicketID: created.ID,
				Mode:     mode,
			})

			if err == nil {
				t.Fatal("expected STATE_CONFLICT error")
			}
			toolErr, ok := err.(*ToolError)
			if !ok {
				t.Fatalf("expected ToolError, got %T", err)
			}
			if toolErr.Code != ErrorCodeStateConflict {
				t.Errorf("expected STATE_CONFLICT, got: %s", toolErr.Code)
			}
			if output.State != "active" {
				t.Errorf("expected state 'active', got: %s", output.State)
			}
		})
	}
}

func TestHandleSpawnSession_StateOrphaned_ModeNormal(t *testing.T) {
	server, cleanup := setupTestServerWithOrphanedSession(t)
	defer cleanup()

	created, _ := server.Store().Create("Test Ticket", "body")
	_, _ = server.Store().SetSession(created.ID, "claude", "window")

	_, output, err := server.handleSpawnSession(context.Background(), nil, SpawnSessionInput{
		TicketID: created.ID,
		Mode:     "normal",
	})

	if err == nil {
		t.Fatal("expected STATE_CONFLICT error")
	}
	toolErr, ok := err.(*ToolError)
	if !ok {
		t.Fatalf("expected ToolError, got %T", err)
	}
	if toolErr.Code != ErrorCodeStateConflict {
		t.Errorf("expected STATE_CONFLICT, got: %s", toolErr.Code)
	}
	if output.State != "orphaned" {
		t.Errorf("expected state 'orphaned', got: %s", output.State)
	}
}

func TestHandleSpawnSession_StateOrphaned_ModeResume(t *testing.T) {
	server, cleanup := setupTestServerWithOrphanedSession(t)
	defer cleanup()

	created, _ := server.Store().Create("Test Ticket", "body")
	_, _ = server.Store().SetSession(created.ID, "claude", "window")

	_, output, err := server.handleSpawnSession(context.Background(), nil, SpawnSessionInput{
		TicketID: created.ID,
		Mode:     "resume",
	})
	if err != nil {
		t.Fatalf("handleSpawnSession failed: %v", err)
	}

	if !output.Success {
		t.Errorf("expected success, got message: %s", output.Message)
	}
	if output.State != "orphaned" {
		t.Errorf("expected state 'orphaned', got: %s", output.State)
	}
}

func TestHandleSpawnSession_StateOrphaned_ModeFresh(t *testing.T) {
	server, cleanup := setupTestServerWithOrphanedSession(t)
	defer cleanup()

	created, _ := server.Store().Create("Test Ticket", "body")
	_, _ = server.Store().SetSession(created.ID, "claude", "window")

	_, output, err := server.handleSpawnSession(context.Background(), nil, SpawnSessionInput{
		TicketID: created.ID,
		Mode:     "fresh",
	})
	if err != nil {
		t.Fatalf("handleSpawnSession failed: %v", err)
	}

	if !output.Success {
		t.Errorf("expected success, got message: %s", output.Message)
	}
	if output.State != "orphaned" {
		t.Errorf("expected state 'orphaned', got: %s", output.State)
	}
}

func TestHandleSpawnSession_StateEnded_ModeNormal(t *testing.T) {
	server, cleanup := setupTestServerWithMockTmux(t, "")
	defer cleanup()

	created, _ := server.Store().Create("Test Ticket", "body")
	_, _ = server.Store().SetSession(created.ID, "claude", "window")
	_ = server.Store().EndSession(created.ID)

	_, output, err := server.handleSpawnSession(context.Background(), nil, SpawnSessionInput{
		TicketID: created.ID,
		Mode:     "normal",
	})
	if err != nil {
		t.Fatalf("handleSpawnSession failed: %v", err)
	}

	if !output.Success {
		t.Errorf("expected success, got message: %s", output.Message)
	}
	if output.State != "ended" {
		t.Errorf("expected state 'ended', got: %s", output.State)
	}
}

func TestHandleSpawnSession_StateEnded_ModeResume(t *testing.T) {
	server, cleanup := setupTestServerWithMockTmux(t, "")
	defer cleanup()

	created, _ := server.Store().Create("Test Ticket", "body")
	_, _ = server.Store().SetSession(created.ID, "claude", "window")
	_ = server.Store().EndSession(created.ID)

	_, output, err := server.handleSpawnSession(context.Background(), nil, SpawnSessionInput{
		TicketID: created.ID,
		Mode:     "resume",
	})

	if err == nil {
		t.Fatal("expected STATE_CONFLICT error")
	}
	toolErr, ok := err.(*ToolError)
	if !ok {
		t.Fatalf("expected ToolError, got %T", err)
	}
	if toolErr.Code != ErrorCodeStateConflict {
		t.Errorf("expected STATE_CONFLICT, got: %s", toolErr.Code)
	}
	if output.State != "ended" {
		t.Errorf("expected state 'ended', got: %s", output.State)
	}
}

func TestHandleSpawnSession_StateEnded_ModeFresh(t *testing.T) {
	server, cleanup := setupTestServerWithMockTmux(t, "")
	defer cleanup()

	created, _ := server.Store().Create("Test Ticket", "body")
	_, _ = server.Store().SetSession(created.ID, "claude", "window")
	_ = server.Store().EndSession(created.ID)

	_, output, err := server.handleSpawnSession(context.Background(), nil, SpawnSessionInput{
		TicketID: created.ID,
		Mode:     "fresh",
	})
	if err != nil {
		t.Fatalf("handleSpawnSession failed: %v", err)
	}

	if !output.Success {
		t.Errorf("expected success, got message: %s", output.Message)
	}
	if output.State != "ended" {
		t.Errorf("expected state 'ended', got: %s", output.State)
	}
}

func TestHandleSpawnSession_InvalidMode(t *testing.T) {
	server, cleanup := setupTestServerWithMockTmux(t, "")
	defer cleanup()

	created, _ := server.Store().Create("Test Ticket", "body")

	_, _, err := server.handleSpawnSession(context.Background(), nil, SpawnSessionInput{
		TicketID: created.ID,
		Mode:     "invalid",
	})

	if err == nil {
		t.Fatal("expected VALIDATION_ERROR")
	}
	toolErr, ok := err.(*ToolError)
	if !ok {
		t.Fatalf("expected ToolError, got %T", err)
	}
	if toolErr.Code != ErrorCodeValidation {
		t.Errorf("expected VALIDATION_ERROR, got: %s", toolErr.Code)
	}
}

func TestHandleSpawnSession_DefaultMode(t *testing.T) {
	server, cleanup := setupTestServerWithMockTmux(t, "")
	defer cleanup()

	created, _ := server.Store().Create("Test Ticket", "body")

	// Empty mode should default to "normal" and succeed
	_, output, err := server.handleSpawnSession(context.Background(), nil, SpawnSessionInput{
		TicketID: created.ID,
		Mode:     "", // empty mode
	})
	if err != nil {
		t.Fatalf("handleSpawnSession failed: %v", err)
	}

	if !output.Success {
		t.Errorf("expected success, got message: %s", output.Message)
	}
	if output.State != "normal" {
		t.Errorf("expected state 'normal', got: %s", output.State)
	}
}

// setupTestServerWithOrphanedSession creates a test server with a mock tmux manager
// that reports windows do not exist (simulating orphaned state).
func setupTestServerWithOrphanedSession(t *testing.T) (*Server, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "mcp-server-test")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	// Create the prompts directory and default templates for the project
	promptsDir := filepath.Join(tmpDir, ".cortex", "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("create prompts dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(promptsDir, "ticket-agent.md"), []byte(prompt.DefaultTicketAgentPrompt), 0644); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("create ticket-agent.md: %v", err)
	}

	// Create mock runner that reports window does NOT exist
	mockRunner := tmux.NewMockRunner()
	mockRunner.SetWindowExists(false)
	tmuxMgr := tmux.NewManagerWithRunner(mockRunner)

	cfg := &Config{
		TicketID:    "",
		TicketsDir:  tmpDir,
		ProjectPath: tmpDir,
		TmuxSession: "test-session",
		TmuxManager: tmuxMgr,
		CortexdPath: "/mock/cortexd",
	}

	server, err := NewServer(cfg)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("create server: %v", err)
	}

	cleanup := func() {
		_ = os.RemoveAll(tmpDir)
	}

	return server, cleanup
}
