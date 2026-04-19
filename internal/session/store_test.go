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

	sess, err := store.Create("a1b2c3d4-e5f6-7890-abcd-ef0123456789", "claude", "fix-auth-bug")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if sess.SessionID == "" {
		t.Error("SessionID should be set")
	}
	if sess.TicketID != "a1b2c3d4-e5f6-7890-abcd-ef0123456789" {
		t.Errorf("ticket_id mismatch")
	}
	if sess.Agent != "claude" {
		t.Errorf("agent = %q, want %q", sess.Agent, "claude")
	}
	if sess.TmuxWindow != "fix-auth-bug" {
		t.Errorf("tmux_window = %q, want %q", sess.TmuxWindow, "fix-auth-bug")
	}
	if sess.Status != AgentStatusStarting {
		t.Errorf("status = %q, want %q", sess.Status, AgentStatusStarting)
	}
	if sess.StartedAt.IsZero() {
		t.Error("started_at should be set")
	}
}

func TestGetBySessionID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	sess, err := store.Create("a1b2c3d4-e5f6-7890-abcd-ef0123456789", "claude", "window")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := store.GetBySessionID(sess.SessionID)
	if err != nil {
		t.Fatalf("GetBySessionID failed: %v", err)
	}

	if got.Agent != "claude" {
		t.Errorf("agent = %q, want %q", got.Agent, "claude")
	}
}

func TestGetByTicketID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticketID := "a1b2c3d4-e5f6-7890-abcd-ef0123456789"
	if _, err := store.Create(ticketID, "claude", "window"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	sess, err := store.GetByTicketID(ticketID)
	if err != nil {
		t.Fatalf("GetByTicketID failed: %v", err)
	}

	if sess.TicketID != ticketID {
		t.Errorf("ticket_id mismatch")
	}
}

func TestGetBySessionIDNotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, err := store.GetBySessionID("nonexistent")
	if !storage.IsNotFound(err) {
		t.Errorf("expected NotFoundError, got %T", err)
	}
}

func TestUpdateStatusBySessionID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	sess, _ := store.Create("a1b2c3d4-e5f6-7890-abcd-ef0123456789", "claude", "window")

	tool := "Edit"
	if err := store.UpdateStatusBySessionID(sess.SessionID, AgentStatusWorking, &tool, nil); err != nil {
		t.Fatalf("UpdateStatusBySessionID failed: %v", err)
	}

	got, _ := store.GetBySessionID(sess.SessionID)
	if got.Status != AgentStatusWorking {
		t.Errorf("status = %q, want %q", got.Status, AgentStatusWorking)
	}
	if got.Tool == nil || *got.Tool != "Edit" {
		t.Error("tool not set correctly")
	}
}

func TestUpdateStatusBySessionIDNotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.UpdateStatusBySessionID("nonexistent", AgentStatusWorking, nil, nil)
	if !storage.IsNotFound(err) {
		t.Errorf("expected NotFoundError, got %T", err)
	}
}

func TestEndBySessionID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	sess, _ := store.Create("a1b2c3d4-e5f6-7890-abcd-ef0123456789", "claude", "window")

	if err := store.EndBySessionID(sess.SessionID); err != nil {
		t.Fatalf("End failed: %v", err)
	}

	_, err := store.GetBySessionID(sess.SessionID)
	if !storage.IsNotFound(err) {
		t.Error("session should be deleted after End")
	}
}

func TestEndByTicketID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticketID := "a1b2c3d4-e5f6-7890-abcd-ef0123456789"
	if _, err := store.Create(ticketID, "claude", "window"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := store.EndByTicketID(ticketID); err != nil {
		t.Fatalf("EndByTicketID failed: %v", err)
	}
	if _, err := store.GetByTicketID(ticketID); !storage.IsNotFound(err) {
		t.Errorf("expected NotFound after EndByTicketID, got %v", err)
	}
}

func TestList(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, _ = store.Create("a1b2c3d4-0000-0000-0000-000000000001", "claude", "window1")
	_, _ = store.Create("b2c3d4e5-0000-0000-0000-000000000002", "opencode", "window2")

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

func TestCreateSetsTicketType(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	sess, err := store.Create("a1b2c3d4-e5f6-7890-abcd-ef0123456789", "claude", "window")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if sess.Type != SessionTypeTicket {
		t.Errorf("Type = %q, want %q", sess.Type, SessionTypeTicket)
	}
	if sess.TicketID != "a1b2c3d4-e5f6-7890-abcd-ef0123456789" {
		t.Errorf("TicketID = %q, want full ticket ID", sess.TicketID)
	}
}

func TestCreateArchitectSetsType(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	sess, err := store.CreateArchitect("claude", "architect-window")
	if err != nil {
		t.Fatalf("CreateArchitect failed: %v", err)
	}

	if sess.Type != SessionTypeArchitect {
		t.Errorf("Type = %q, want %q", sess.Type, SessionTypeArchitect)
	}
	if sess.TicketID != "" {
		t.Errorf("TicketID = %q, want empty for architect session", sess.TicketID)
	}
}

func TestCreateArchitectReplacesExisting(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	first, _ := store.CreateArchitect("claude", "w1")
	second, _ := store.CreateArchitect("codex", "w2")

	if first.SessionID == second.SessionID {
		t.Error("architect session IDs should differ across recreations")
	}

	sessions, _ := store.List()
	archCount := 0
	for _, s := range sessions {
		if s.Type == SessionTypeArchitect {
			archCount++
		}
	}
	if archCount != 1 {
		t.Errorf("want exactly 1 architect session, got %d", archCount)
	}
}

func TestConcurrentAccess(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	sess, _ := store.Create("a1b2c3d4-e5f6-7890-abcd-ef0123456789", "claude", "window")

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			tool := "Read"
			_ = store.UpdateStatusBySessionID(sess.SessionID, AgentStatusWorking, &tool, nil)
		}()
	}

	wg.Wait()

	got, err := store.GetBySessionID(sess.SessionID)
	if err != nil {
		t.Fatalf("Get after concurrent access failed: %v", err)
	}
	if got.Status != AgentStatusWorking {
		t.Errorf("status = %q, want %q", got.Status, AgentStatusWorking)
	}
}
