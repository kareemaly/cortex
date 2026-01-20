package mcp

import (
	"context"
	"os"
	"testing"

	"github.com/kareemaly/cortex1/internal/ticket"
)

// Architect tool tests

func TestHandleListTickets(t *testing.T) {
	server, cleanup := setupTestServer(t, "")
	defer cleanup()

	// Create some tickets
	_, _ = server.Store().Create("Ticket 1", "body 1")
	_, _ = server.Store().Create("Ticket 2", "body 2")

	// List all tickets
	_, output, err := server.handleListTickets(context.Background(), nil, ListTicketsInput{})
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

func TestHandleSearchTickets(t *testing.T) {
	server, cleanup := setupTestServer(t, "")
	defer cleanup()

	// Create tickets
	_, _ = server.Store().Create("Fix login bug", "Authentication issue")
	_, _ = server.Store().Create("Add feature", "New feature")

	// Search for "login"
	_, output, err := server.handleSearchTickets(context.Background(), nil, SearchTicketsInput{
		Query: "login",
	})
	if err != nil {
		t.Fatalf("handleSearchTickets failed: %v", err)
	}

	if output.Total != 1 {
		t.Errorf("total = %d, want 1", output.Total)
	}
}

func TestHandleSearchTicketsEmptyQuery(t *testing.T) {
	server, cleanup := setupTestServer(t, "")
	defer cleanup()

	_, _, err := server.handleSearchTickets(context.Background(), nil, SearchTicketsInput{
		Query: "",
	})

	if err == nil {
		t.Error("expected error for empty query")
	}
	if !IsToolError(err) {
		t.Errorf("expected ToolError, got %T", err)
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
	server, cleanup := setupTestServer(t, "")
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

	// Output should have a message (either success or tmux not available)
	if output.Message == "" {
		t.Error("expected message")
	}

	// Should return the ticket ID
	if output.TicketID != created.ID {
		t.Errorf("expected ticket_id %s, got %s", created.ID, output.TicketID)
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
	_, err = server.Store().AddSession(created.ID, "claude", "window", nil)
	if err != nil {
		t.Fatalf("failed to add session: %v", err)
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

func TestHandleGetSessionStatus(t *testing.T) {
	server, cleanup := setupTestServer(t, "")
	defer cleanup()

	// Create ticket with active session
	created, _ := server.Store().Create("Test", "")
	session, _ := server.Store().AddSession(created.ID, "claude", "window", nil)

	_, output, err := server.handleGetSessionStatus(context.Background(), nil, GetSessionStatusInput{
		TicketID: created.ID,
	})
	if err != nil {
		t.Fatalf("handleGetSessionStatus failed: %v", err)
	}

	if output.Session == nil {
		t.Fatal("session should not be nil")
	}
	if output.Session.ID != session.ID {
		t.Errorf("session ID = %q, want %q", output.Session.ID, session.ID)
	}
}

func TestHandleGetSessionStatusNoActive(t *testing.T) {
	server, cleanup := setupTestServer(t, "")
	defer cleanup()

	// Create ticket without session
	created, _ := server.Store().Create("Test", "")

	_, output, err := server.handleGetSessionStatus(context.Background(), nil, GetSessionStatusInput{
		TicketID: created.ID,
	})
	if err != nil {
		t.Fatalf("handleGetSessionStatus failed: %v", err)
	}

	if output.Session != nil {
		t.Error("session should be nil when no active session")
	}
	if output.Message == "" {
		t.Error("expected message about no active session")
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

	// Add a session to the ticket
	_, err = store.AddSession(tk.ID, "claude", "window", nil)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("add session: %v", err)
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

func TestHandlePickupTicket(t *testing.T) {
	server, _, cleanup := setupTicketSession(t)
	defer cleanup()

	_, output, err := server.handlePickupTicket(context.Background(), nil, EmptyInput{})
	if err != nil {
		t.Fatalf("handlePickupTicket failed: %v", err)
	}

	if !output.Success {
		t.Error("pickup should succeed")
	}

	// Verify ticket moved to progress
	tk, status, _ := server.Store().Get(server.Session().TicketID)
	if status != ticket.StatusProgress {
		t.Errorf("status = %q, want %q", status, ticket.StatusProgress)
	}
	_ = tk
}

func TestHandlePickupTicketAlreadyInProgress(t *testing.T) {
	server, _, cleanup := setupTicketSession(t)
	defer cleanup()

	// Move to progress first
	_ = server.Store().Move(server.Session().TicketID, ticket.StatusProgress)

	_, output, err := server.handlePickupTicket(context.Background(), nil, EmptyInput{})
	if err != nil {
		t.Fatalf("handlePickupTicket failed: %v", err)
	}

	if !output.Success {
		t.Error("should still succeed")
	}
	if output.Message != "Ticket is already in progress" {
		t.Errorf("unexpected message: %q", output.Message)
	}
}

func TestHandleSubmitReport(t *testing.T) {
	server, _, cleanup := setupTicketSession(t)
	defer cleanup()

	scopeChange := "Added new feature"
	_, output, err := server.handleSubmitReport(context.Background(), nil, SubmitReportInput{
		Files:        []string{"file1.go", "file2.go"},
		ScopeChanges: &scopeChange,
		Decisions:    []string{"Decision 1"},
		Summary:      "Completed work",
	})
	if err != nil {
		t.Fatalf("handleSubmitReport failed: %v", err)
	}

	if !output.Success {
		t.Error("submit should succeed")
	}
	if len(output.Report.Files) != 2 {
		t.Errorf("files count = %d, want 2", len(output.Report.Files))
	}
}

func TestHandleApprove(t *testing.T) {
	server, ticketID, cleanup := setupTicketSession(t)
	defer cleanup()

	_, output, err := server.handleApprove(context.Background(), nil, ApproveInput{
		Summary: "Work completed",
	})
	if err != nil {
		t.Fatalf("handleApprove failed: %v", err)
	}

	if !output.Success {
		t.Error("approve should succeed")
	}
	if output.TicketID != ticketID {
		t.Errorf("ticket ID = %q, want %q", output.TicketID, ticketID)
	}
	if output.Status != "done" {
		t.Errorf("status = %q, want %q", output.Status, "done")
	}

	// Verify ticket is done
	_, status, _ := server.Store().Get(ticketID)
	if status != ticket.StatusDone {
		t.Errorf("ticket status = %q, want %q", status, ticket.StatusDone)
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
