package ticket

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/kareemaly/cortex/internal/storage"
)

func setupTestStore(t *testing.T) (*Store, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "ticket-store-test")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	store, err := NewStore(tmpDir, nil, "")
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

	ticket, err := store.Create("Test Ticket", "Test body", "", nil, nil, nil)
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
	if ticket.Type != DefaultTicketType {
		t.Errorf("type = %q, want %q", ticket.Type, DefaultTicketType)
	}
	if ticket.Created.IsZero() {
		t.Error("created date should be set")
	}
	if ticket.Due != nil {
		t.Error("due date should be nil")
	}
}

func TestStoreCreateEmptyTitle(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, err := store.Create("", "body", "", nil, nil, nil)
	if err == nil {
		t.Error("expected error for empty title")
	}
	if _, ok := err.(*ValidationError); !ok {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestStoreCreateWithReferences(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	refs := []string{"doc:abc123"}
	ticket, err := store.Create("Test", "body", "", nil, refs, nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if len(ticket.References) != 1 || ticket.References[0] != "doc:abc123" {
		t.Errorf("references = %v, want [doc:abc123]", ticket.References)
	}
}

func TestStoreGet(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	created, err := store.Create("Test Ticket", "body", "", nil, nil, nil)
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
	if retrieved.Title != "Test Ticket" {
		t.Errorf("title = %q, want %q", retrieved.Title, "Test Ticket")
	}
	if retrieved.Body != "body" {
		t.Errorf("body = %q, want %q", retrieved.Body, "body")
	}
}

func TestStoreGetLoadsComments(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticket, _ := store.Create("Test Ticket", "body", "", nil, nil, nil)
	_, _ = store.AddComment(ticket.ID, "claude", CommentGeneral, "comment 1", nil)
	_, _ = store.AddComment(ticket.ID, "claude", CommentGeneral, "comment 2", nil)

	retrieved, _, err := store.Get(ticket.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if len(retrieved.Comments) != 2 {
		t.Errorf("comments count = %d, want 2", len(retrieved.Comments))
	}
}

func TestStoreListDoesNotLoadComments(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticket, _ := store.Create("Test Ticket", "body", "", nil, nil, nil)
	_, _ = store.AddComment(ticket.ID, "claude", CommentGeneral, "comment", nil)

	tickets, err := store.List(StatusBacklog)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(tickets) != 1 {
		t.Fatalf("len(tickets) = %d, want 1", len(tickets))
	}
	if len(tickets[0].Comments) != 0 {
		t.Errorf("List should not load comments, got %d", len(tickets[0].Comments))
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

	ticket, _ := store.Create("Original Title", "Original body", "", nil, nil, nil)

	newTitle := "Updated Title"
	newBody := "Updated body"
	updated, err := store.Update(ticket.ID, &newTitle, &newBody, nil, nil)
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

	ticket, _ := store.Create("Original Title", "Original body", "", nil, nil, nil)

	newTitle := "Updated Title"
	updated, err := store.Update(ticket.ID, &newTitle, nil, nil, nil)
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

func TestStoreUpdateTitleRename(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticket, _ := store.Create("Old Title", "body", "", nil, nil, nil)
	shortID := storage.ShortID(ticket.ID)

	// Verify old directory exists
	oldDir := filepath.Join(store.ticketsDir, "backlog", "old-title-"+shortID)
	if _, err := os.Stat(oldDir); err != nil {
		t.Fatalf("old dir should exist: %v", err)
	}

	newTitle := "New Title"
	_, err := store.Update(ticket.ID, &newTitle, nil, nil, nil)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Old dir should be gone
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Error("old dir should be removed after title change")
	}

	// New dir should exist
	newDir := filepath.Join(store.ticketsDir, "backlog", "new-title-"+shortID)
	if _, err := os.Stat(newDir); err != nil {
		t.Errorf("new dir should exist: %v", err)
	}

	// Should still be retrievable
	retrieved, _, err := store.Get(ticket.ID)
	if err != nil {
		t.Fatalf("Get after rename failed: %v", err)
	}
	if retrieved.Title != "New Title" {
		t.Errorf("title = %q, want %q", retrieved.Title, "New Title")
	}
}

func TestStoreDelete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticket, _ := store.Create("Test Ticket", "body", "", nil, nil, nil)

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

	_, _ = store.Create("Ticket 1", "", "", nil, nil, nil)
	_, _ = store.Create("Ticket 2", "", "", nil, nil, nil)

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

	t1, _ := store.Create("Backlog Ticket", "", "", nil, nil, nil)
	t2, _ := store.Create("Progress Ticket", "", "", nil, nil, nil)
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

	ticket, _ := store.Create("Test Ticket", "body", "", nil, nil, nil)

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

func TestStoreMoveSameStatus(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticket, _ := store.Create("Test Ticket", "body", "", nil, nil, nil)

	if err := store.Move(ticket.ID, StatusBacklog); err != nil {
		t.Fatalf("Move to same status failed: %v", err)
	}

	_, status, err := store.Get(ticket.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if status != StatusBacklog {
		t.Errorf("status = %q, want %q", status, StatusBacklog)
	}
}

func TestStoreAddComment(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticket, _ := store.Create("Test Ticket", "body", "", nil, nil, nil)

	comment, err := store.AddComment(ticket.ID, "claude", CommentGeneral, "Test comment", nil)
	if err != nil {
		t.Fatalf("AddComment failed: %v", err)
	}

	if comment.ID == "" {
		t.Error("comment ID should not be empty")
	}
	if comment.Type != CommentGeneral {
		t.Errorf("type = %q, want %q", comment.Type, CommentGeneral)
	}
	if comment.Content != "Test comment" {
		t.Errorf("content = %q, want %q", comment.Content, "Test comment")
	}
	if comment.Author != "claude" {
		t.Errorf("author = %q, want %q", comment.Author, "claude")
	}

	retrieved, _, _ := store.Get(ticket.ID)
	if len(retrieved.Comments) != 1 {
		t.Errorf("comments count = %d, want 1", len(retrieved.Comments))
	}
}

func TestStoreAddCommentWithAction(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticket, _ := store.Create("Test Ticket", "body", "", nil, nil, nil)

	action := &storage.CommentAction{
		Type: "git_diff",
		Args: storage.GitDiffArgs{RepoPath: "/path/to/repo", Commit: "abc123"},
	}
	comment, err := store.AddComment(ticket.ID, "claude", CommentReviewRequested, "Review changes", action)
	if err != nil {
		t.Fatalf("AddComment failed: %v", err)
	}

	if comment.Action == nil {
		t.Fatal("action should not be nil")
	}
	if comment.Action.Type != "git_diff" {
		t.Errorf("action type = %q, want %q", comment.Action.Type, "git_diff")
	}
}

func TestStoreAddCommentValidation(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	tk, _ := store.Create("Test Ticket", "body", "", nil, nil, nil)

	_, err := store.AddComment(tk.ID, "claude", CommentGeneral, "", nil)
	if err == nil {
		t.Error("expected error for empty content")
	}
	if _, ok := err.(*ValidationError); !ok {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestStoreListComments(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticket, _ := store.Create("Test Ticket", "body", "", nil, nil, nil)
	_, _ = store.AddComment(ticket.ID, "claude", CommentGeneral, "First", nil)
	_, _ = store.AddComment(ticket.ID, "claude", CommentGeneral, "Second", nil)

	comments, err := store.ListComments(ticket.ID)
	if err != nil {
		t.Fatalf("ListComments failed: %v", err)
	}

	if len(comments) != 2 {
		t.Fatalf("len(comments) = %d, want 2", len(comments))
	}

	// Should be sorted by created time
	if comments[0].Content != "First" {
		t.Errorf("first comment = %q, want %q", comments[0].Content, "First")
	}
}

func TestStoreConcurrentUpdates(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	tk, err := store.Create("Concurrent Ticket", "initial body", "", nil, nil, nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	const goroutines = 10
	const updatesPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(g int) {
			defer wg.Done()
			for i := 0; i < updatesPerGoroutine; i++ {
				title := fmt.Sprintf("Title-%d-%d", g, i)
				body := fmt.Sprintf("Body-%d-%d", g, i)
				_, err := store.Update(tk.ID, &title, &body, nil, nil)
				if err != nil {
					t.Errorf("Update goroutine %d iter %d failed: %v", g, i, err)
					return
				}
			}
		}(g)
	}

	wg.Wait()

	retrieved, status, err := store.Get(tk.ID)
	if err != nil {
		t.Fatalf("Get after concurrent updates failed: %v", err)
	}
	if status != StatusBacklog {
		t.Errorf("status = %q, want %q", status, StatusBacklog)
	}
	if retrieved.Title == "" {
		t.Error("title should not be empty after concurrent updates")
	}
}

func TestStoreConcurrentAddComments(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	tk, err := store.Create("Comment Ticket", "body", "", nil, nil, nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	const goroutines = 10

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(g int) {
			defer wg.Done()
			content := fmt.Sprintf("Comment from goroutine %d", g)
			_, err := store.AddComment(tk.ID, "claude", CommentGeneral, content, nil)
			if err != nil {
				t.Errorf("AddComment goroutine %d failed: %v", g, err)
			}
		}(g)
	}

	wg.Wait()

	// With file-per-comment, concurrent adds should not lose comments
	retrieved, _, err := store.Get(tk.ID)
	if err != nil {
		t.Fatalf("Get after concurrent comments failed: %v", err)
	}
	if len(retrieved.Comments) != goroutines {
		t.Errorf("comments count = %d, want %d", len(retrieved.Comments), goroutines)
	}
}

func TestStoreDirLayout(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticket, _ := store.Create("Fix Auth Bug", "body", "", nil, nil, nil)
	shortID := storage.ShortID(ticket.ID)

	// Verify directory structure
	entityDir := filepath.Join(store.ticketsDir, "backlog", "fix-auth-bug-"+shortID)
	if _, err := os.Stat(entityDir); err != nil {
		t.Fatalf("entity dir should exist: %v", err)
	}

	indexPath := filepath.Join(entityDir, "index.md")
	if _, err := os.Stat(indexPath); err != nil {
		t.Fatalf("index.md should exist: %v", err)
	}
}

func TestStoreSetDueDate(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticket, _ := store.Create("Test", "body", "", nil, nil, nil)

	due := ticket.Created.AddDate(0, 0, 7)
	updated, err := store.SetDueDate(ticket.ID, &due)
	if err != nil {
		t.Fatalf("SetDueDate failed: %v", err)
	}

	if updated.Due == nil || !updated.Due.Equal(due) {
		t.Error("due date should be set")
	}
}

func TestStoreClearDueDate(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	due := store.ticketsDir // irrelevant, just need a time
	_ = due
	ticket, _ := store.Create("Test", "body", "", nil, nil, nil)

	d := ticket.Created.AddDate(0, 0, 7)
	_, _ = store.SetDueDate(ticket.ID, &d)

	updated, err := store.ClearDueDate(ticket.ID)
	if err != nil {
		t.Fatalf("ClearDueDate failed: %v", err)
	}

	if updated.Due != nil {
		t.Error("due date should be cleared")
	}
}

func TestStoreStatusConstants(t *testing.T) {
	if StatusBacklog != "backlog" {
		t.Error("StatusBacklog should be 'backlog'")
	}
	if StatusProgress != "progress" {
		t.Error("StatusProgress should be 'progress'")
	}
	if StatusReview != "review" {
		t.Error("StatusReview should be 'review'")
	}
	if StatusDone != "done" {
		t.Error("StatusDone should be 'done'")
	}
}
