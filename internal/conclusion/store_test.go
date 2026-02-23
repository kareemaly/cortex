package conclusion

import (
	"os"
	"testing"
)

func TestCreateAndGet(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir, nil, "/test/project")
	if err != nil {
		t.Fatal(err)
	}

	c, err := store.Create("work", "ticket-123", "/repo", "# Summary\n\nDid some work.")
	if err != nil {
		t.Fatal(err)
	}

	if c.ID == "" {
		t.Error("expected non-empty ID")
	}
	if c.Type != TypeWork {
		t.Errorf("expected type work, got %s", c.Type)
	}
	if c.Ticket != "ticket-123" {
		t.Errorf("expected ticket ticket-123, got %s", c.Ticket)
	}
	if c.Repo != "/repo" {
		t.Errorf("expected repo /repo, got %s", c.Repo)
	}

	// Get by ID
	got, err := store.Get(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != c.ID {
		t.Errorf("expected ID %s, got %s", c.ID, got.ID)
	}
	if got.Body != c.Body {
		t.Errorf("expected body %q, got %q", c.Body, got.Body)
	}
}

func TestList(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir, nil, "/test/project")
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Create("work", "t1", "", "First")
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Create("research", "t2", "", "Second")
	if err != nil {
		t.Fatal(err)
	}

	list, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 conclusions, got %d", len(list))
	}
}

func TestGetNotFound(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir, nil, "/test/project")
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Get("nonexistent")
	if !IsNotFound(err) {
		t.Errorf("expected NotFoundError, got %v", err)
	}
}

func TestCreateValidation(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir, nil, "/test/project")
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Create("work", "", "", "")
	if err == nil {
		t.Error("expected error for empty body")
	}
}

func TestNewStoreCreatesDir(t *testing.T) {
	dir := t.TempDir()
	sessDir := dir + "/sessions"

	_, err := NewStore(sessDir, nil, "/test/project")
	if err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(sessDir)
	if err != nil {
		t.Fatal("sessions dir should exist:", err)
	}
	if !info.IsDir() {
		t.Fatal("sessions path should be a directory")
	}
}
