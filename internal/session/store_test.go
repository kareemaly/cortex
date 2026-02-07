package session

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/kareemaly/cortex/internal/storage"
)

func setupTestStore(t *testing.T) (*Store, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "session-store-test")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	path := filepath.Join(dir, "sessions.json")
	store := NewStore(path)

	return store, func() { _ = os.RemoveAll(dir) }
}

func TestCreate(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	key, session, err := store.Create("a1b2c3d4-e5f6-7890-abcd-ef0123456789", "claude", "fix-auth-bug", nil, nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if key != "a1b2c3d4" {
		t.Errorf("key = %q, want %q", key, "a1b2c3d4")
	}
	if session.TicketID != "a1b2c3d4-e5f6-7890-abcd-ef0123456789" {
		t.Errorf("ticket_id mismatch")
	}
	if session.Agent != "claude" {
		t.Errorf("agent = %q, want %q", session.Agent, "claude")
	}
	if session.TmuxWindow != "fix-auth-bug" {
		t.Errorf("tmux_window = %q, want %q", session.TmuxWindow, "fix-auth-bug")
	}
	if session.Status != AgentStatusStarting {
		t.Errorf("status = %q, want %q", session.Status, AgentStatusStarting)
	}
	if session.StartedAt.IsZero() {
		t.Error("started_at should be set")
	}
}

func TestCreateWithOptionalFields(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wt := "/path/to/worktree"
	fb := "ticket/fix-auth"
	_, session, err := store.Create("a1b2c3d4-e5f6-7890-abcd-ef0123456789", "claude", "window", &wt, &fb)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if session.WorktreePath == nil || *session.WorktreePath != wt {
		t.Error("worktree_path not set correctly")
	}
	if session.FeatureBranch == nil || *session.FeatureBranch != fb {
		t.Error("feature_branch not set correctly")
	}
}

func TestGet(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	key, _, err := store.Create("a1b2c3d4-e5f6-7890-abcd-ef0123456789", "claude", "window", nil, nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	session, err := store.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if session.Agent != "claude" {
		t.Errorf("agent = %q, want %q", session.Agent, "claude")
	}
}

func TestGetByTicketID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticketID := "a1b2c3d4-e5f6-7890-abcd-ef0123456789"
	_, _, err := store.Create(ticketID, "claude", "window", nil, nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	session, err := store.GetByTicketID(ticketID)
	if err != nil {
		t.Fatalf("GetByTicketID failed: %v", err)
	}

	if session.TicketID != ticketID {
		t.Errorf("ticket_id mismatch")
	}
}

func TestGetNotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, err := store.Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
	if !storage.IsNotFound(err) {
		t.Errorf("expected NotFoundError, got %T", err)
	}
}

func TestUpdateStatus(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	key, _, _ := store.Create("a1b2c3d4-e5f6-7890-abcd-ef0123456789", "claude", "window", nil, nil)

	tool := "Edit"
	err := store.UpdateStatus(key, AgentStatusInProgress, &tool)
	if err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	session, _ := store.Get(key)
	if session.Status != AgentStatusInProgress {
		t.Errorf("status = %q, want %q", session.Status, AgentStatusInProgress)
	}
	if session.Tool == nil || *session.Tool != "Edit" {
		t.Error("tool not set correctly")
	}
}

func TestUpdateStatusNotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.UpdateStatus("nonexistent", AgentStatusInProgress, nil)
	if !storage.IsNotFound(err) {
		t.Errorf("expected NotFoundError, got %T", err)
	}
}

func TestEnd(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	key, _, _ := store.Create("a1b2c3d4-e5f6-7890-abcd-ef0123456789", "claude", "window", nil, nil)

	err := store.End(key)
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}

	_, err = store.Get(key)
	if !storage.IsNotFound(err) {
		t.Error("session should be deleted after End")
	}
}

func TestEndNotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.End("nonexistent")
	if !storage.IsNotFound(err) {
		t.Errorf("expected NotFoundError, got %T", err)
	}
}

func TestList(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, _, _ = store.Create("a1b2c3d4-0000-0000-0000-000000000001", "claude", "window1", nil, nil)
	_, _, _ = store.Create("b2c3d4e5-0000-0000-0000-000000000002", "copilot", "window2", nil, nil)

	sessions, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("len(sessions) = %d, want 2", len(sessions))
	}
}

func TestMissingFile(t *testing.T) {
	store := NewStore("/nonexistent/path/sessions.json")

	sessions, err := store.List()
	if err != nil {
		t.Fatalf("List should handle missing file: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("len(sessions) = %d, want 0", len(sessions))
	}
}

func TestEmptyFile(t *testing.T) {
	dir, err := os.MkdirTemp("", "session-test")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	path := filepath.Join(dir, "sessions.json")
	_ = os.WriteFile(path, []byte(""), 0644)

	store := NewStore(path)
	sessions, err := store.List()
	if err != nil {
		t.Fatalf("List should handle empty file: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("len(sessions) = %d, want 0", len(sessions))
	}
}

func TestConcurrentAccess(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Create a session to operate on
	key, _, _ := store.Create("a1b2c3d4-e5f6-7890-abcd-ef0123456789", "claude", "window", nil, nil)

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			tool := "Read"
			_ = store.UpdateStatus(key, AgentStatusInProgress, &tool)
		}()
	}

	wg.Wait()

	session, err := store.Get(key)
	if err != nil {
		t.Fatalf("Get after concurrent access failed: %v", err)
	}
	if session.Status != AgentStatusInProgress {
		t.Errorf("status = %q, want %q", session.Status, AgentStatusInProgress)
	}
}
