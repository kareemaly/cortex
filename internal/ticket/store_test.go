package ticket

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestStore(t *testing.T) (*Store, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "ticket-store-test")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	store, err := NewStore(tmpDir)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("create store: %v", err)
	}

	cleanup := func() {
		_ = os.RemoveAll(tmpDir)
	}

	return store, cleanup
}

func TestNewStore(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Verify directories were created
	for _, status := range []Status{StatusBacklog, StatusProgress, StatusReview, StatusDone} {
		dir := filepath.Join(store.ticketsDir, string(status))
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("directory %s not created: %v", status, err)
		}
		if !info.IsDir() {
			t.Errorf("%s is not a directory", status)
		}
	}
}

func TestStoreCreate(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticket, err := store.Create("Test Ticket", "Test body")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if ticket.ID == "" {
		t.Error("ticket ID should not be empty")
	}
	if ticket.Title != "Test Ticket" {
		t.Errorf("title = %q, want %q", ticket.Title, "Test Ticket")
	}
	if ticket.Body != "Test body" {
		t.Errorf("body = %q, want %q", ticket.Body, "Test body")
	}
	if ticket.Dates.Created.IsZero() {
		t.Error("created date should be set")
	}
	if ticket.Dates.Done != nil {
		t.Error("done date should be nil")
	}
}

func TestStoreCreateEmptyTitle(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, err := store.Create("", "body")
	if err == nil {
		t.Error("expected error for empty title")
	}
	if _, ok := err.(*ValidationError); !ok {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestStoreGet(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	created, err := store.Create("Test Ticket", "body")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	retrieved, status, err := store.Get(created.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if status != StatusBacklog {
		t.Errorf("status = %q, want %q", status, StatusBacklog)
	}
	if retrieved.ID != created.ID {
		t.Errorf("ID = %q, want %q", retrieved.ID, created.ID)
	}
}

func TestStoreGetNotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, _, err := store.Get("nonexistent-id")
	if err == nil {
		t.Error("expected error for nonexistent ticket")
	}
	if !IsNotFound(err) {
		t.Errorf("expected NotFoundError, got %T", err)
	}
}

func TestStoreUpdate(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticket, _ := store.Create("Original Title", "Original body")

	newTitle := "Updated Title"
	newBody := "Updated body"
	updated, err := store.Update(ticket.ID, &newTitle, &newBody)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.Title != newTitle {
		t.Errorf("title = %q, want %q", updated.Title, newTitle)
	}
	if updated.Body != newBody {
		t.Errorf("body = %q, want %q", updated.Body, newBody)
	}
}

func TestStoreUpdatePartial(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticket, _ := store.Create("Original Title", "Original body")

	newTitle := "Updated Title"
	updated, err := store.Update(ticket.ID, &newTitle, nil)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.Title != newTitle {
		t.Errorf("title = %q, want %q", updated.Title, newTitle)
	}
	if updated.Body != "Original body" {
		t.Errorf("body should remain unchanged")
	}
}

func TestStoreDelete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticket, _ := store.Create("Test Ticket", "body")

	if err := store.Delete(ticket.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, _, err := store.Get(ticket.ID)
	if !IsNotFound(err) {
		t.Error("ticket should not exist after delete")
	}
}

func TestStoreList(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, _ = store.Create("Ticket 1", "")
	_, _ = store.Create("Ticket 2", "")

	tickets, err := store.List(StatusBacklog)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(tickets) != 2 {
		t.Errorf("len(tickets) = %d, want 2", len(tickets))
	}
}

func TestStoreListAll(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	t1, _ := store.Create("Backlog Ticket", "")
	t2, _ := store.Create("Progress Ticket", "")
	_ = store.Move(t2.ID, StatusProgress)

	all, err := store.ListAll()
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	if len(all[StatusBacklog]) != 1 {
		t.Errorf("backlog count = %d, want 1", len(all[StatusBacklog]))
	}
	if len(all[StatusProgress]) != 1 {
		t.Errorf("progress count = %d, want 1", len(all[StatusProgress]))
	}
	if all[StatusBacklog][0].ID != t1.ID {
		t.Error("wrong ticket in backlog")
	}
}

func TestStoreMove(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticket, _ := store.Create("Test Ticket", "body")

	if err := store.Move(ticket.ID, StatusProgress); err != nil {
		t.Fatalf("Move failed: %v", err)
	}

	_, status, err := store.Get(ticket.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if status != StatusProgress {
		t.Errorf("status = %q, want %q", status, StatusProgress)
	}
}

func TestStoreMoveToDoneSetsDoneDate(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticket, _ := store.Create("Test Ticket", "body")

	if err := store.Move(ticket.ID, StatusDone); err != nil {
		t.Fatalf("Move failed: %v", err)
	}

	retrieved, _, _ := store.Get(ticket.ID)
	if retrieved.Dates.Done == nil {
		t.Error("done date should be set when moving to done")
	}
}

func TestStoreMoveToProgressSetsProgressDate(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticket, _ := store.Create("Test Ticket", "body")

	if err := store.Move(ticket.ID, StatusProgress); err != nil {
		t.Fatalf("Move failed: %v", err)
	}

	retrieved, _, _ := store.Get(ticket.ID)
	if retrieved.Dates.Progress == nil {
		t.Error("progress date should be set when moving to progress")
	}
}

func TestStoreMoveToReviewSetsReviewedDate(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticket, _ := store.Create("Test Ticket", "body")
	_ = store.Move(ticket.ID, StatusProgress)

	if err := store.Move(ticket.ID, StatusReview); err != nil {
		t.Fatalf("Move failed: %v", err)
	}

	retrieved, _, _ := store.Get(ticket.ID)
	if retrieved.Dates.Reviewed == nil {
		t.Error("reviewed date should be set when moving to review")
	}
}

func TestStoreSetSession(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticket, _ := store.Create("Test Ticket", "body")

	session, err := store.SetSession(ticket.ID, "claude", "test-window", nil, nil)
	if err != nil {
		t.Fatalf("SetSession failed: %v", err)
	}

	if session.ID == "" {
		t.Error("session ID should not be empty")
	}
	if session.Agent != "claude" {
		t.Errorf("agent = %q, want %q", session.Agent, "claude")
	}
	if !session.IsActive() {
		t.Error("new session should be active")
	}

	retrieved, _, _ := store.Get(ticket.ID)
	if retrieved.Session == nil {
		t.Error("session should not be nil")
	}
	if retrieved.Session.ID != session.ID {
		t.Errorf("session ID = %q, want %q", retrieved.Session.ID, session.ID)
	}
}

func TestStoreEndSession(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticket, _ := store.Create("Test Ticket", "body")
	_, _ = store.SetSession(ticket.ID, "claude", "window", nil, nil)

	if err := store.EndSession(ticket.ID); err != nil {
		t.Fatalf("EndSession failed: %v", err)
	}

	retrieved, _, _ := store.Get(ticket.ID)
	if retrieved.Session.IsActive() {
		t.Error("session should not be active after ending")
	}
}

func TestStoreUpdateSessionStatus(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticket, _ := store.Create("Test Ticket", "body")
	_, _ = store.SetSession(ticket.ID, "claude", "window", nil, nil)

	tool := "Edit"
	work := "Writing code"
	err := store.UpdateSessionStatus(ticket.ID, AgentStatusInProgress, &tool, &work)
	if err != nil {
		t.Fatalf("UpdateSessionStatus failed: %v", err)
	}

	retrieved, _, _ := store.Get(ticket.ID)
	if retrieved.Session.CurrentStatus.Status != AgentStatusInProgress {
		t.Error("status should be updated")
	}
	if len(retrieved.Session.StatusHistory) != 2 {
		t.Errorf("history count = %d, want 2", len(retrieved.Session.StatusHistory))
	}
}

func TestStoreAddComment(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticket, _ := store.Create("Test Ticket", "body")
	session, _ := store.SetSession(ticket.ID, "claude", "window", nil, nil)

	comment, err := store.AddComment(ticket.ID, session.ID, CommentDecision, "Test decision")
	if err != nil {
		t.Fatalf("AddComment failed: %v", err)
	}

	if comment.ID == "" {
		t.Error("comment ID should not be empty")
	}
	if comment.Type != CommentDecision {
		t.Errorf("type = %q, want %q", comment.Type, CommentDecision)
	}
	if comment.Content != "Test decision" {
		t.Errorf("content = %q, want %q", comment.Content, "Test decision")
	}
	if comment.SessionID != session.ID {
		t.Errorf("session_id = %q, want %q", comment.SessionID, session.ID)
	}

	retrieved, _, _ := store.Get(ticket.ID)
	if len(retrieved.Comments) != 1 {
		t.Errorf("comments count = %d, want 1", len(retrieved.Comments))
	}
}

func TestStoreSessionNotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticket, _ := store.Create("Test Ticket", "body")

	// Ticket has no session, so EndSession should return NotFoundError
	err := store.EndSession(ticket.ID)
	if !IsNotFound(err) {
		t.Errorf("expected NotFoundError, got %T", err)
	}
}
