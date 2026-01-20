package mcp

import (
	"os"
	"testing"
)

func setupTestServer(t *testing.T, ticketID string) (*Server, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "mcp-server-test")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	cfg := &Config{
		TicketID:   ticketID,
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

	return server, cleanup
}

func TestNewServerArchitect(t *testing.T) {
	server, cleanup := setupTestServer(t, "")
	defer cleanup()

	if !server.IsArchitectSession() {
		t.Error("expected architect session")
	}
	if server.IsTicketSession() {
		t.Error("should not be ticket session")
	}
	if server.Session().Type != SessionTypeArchitect {
		t.Errorf("session type = %q, want %q", server.Session().Type, SessionTypeArchitect)
	}
	if server.Session().TicketID != "" {
		t.Error("architect session should not have ticket ID")
	}
}

func TestNewServerTicket(t *testing.T) {
	server, cleanup := setupTestServer(t, "test-ticket-123")
	defer cleanup()

	if server.IsArchitectSession() {
		t.Error("should not be architect session")
	}
	if !server.IsTicketSession() {
		t.Error("expected ticket session")
	}
	if server.Session().Type != SessionTypeTicket {
		t.Errorf("session type = %q, want %q", server.Session().Type, SessionTypeTicket)
	}
	if server.Session().TicketID != "test-ticket-123" {
		t.Errorf("ticket ID = %q, want %q", server.Session().TicketID, "test-ticket-123")
	}
}

func TestNewServerDefaultConfig(t *testing.T) {
	// Use a temp directory to avoid modifying user's actual ticket store
	tmpDir, err := os.MkdirTemp("", "mcp-default-test")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cfg := &Config{
		TicketsDir: tmpDir,
	}

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	// Should default to architect session
	if !server.IsArchitectSession() {
		t.Error("expected architect session by default")
	}
}

func TestNewServerNilConfig(t *testing.T) {
	// This test depends on the user's home directory being accessible
	// but we create a temp dir to avoid affecting real data
	tmpDir, err := os.MkdirTemp("", "mcp-nil-config-test")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Temporarily set HOME to our temp dir
	oldHome := os.Getenv("HOME")
	if err := os.Setenv("HOME", tmpDir); err != nil {
		t.Fatalf("set HOME: %v", err)
	}
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	server, err := NewServer(nil)
	if err != nil {
		t.Fatalf("NewServer with nil config failed: %v", err)
	}

	if !server.IsArchitectSession() {
		t.Error("expected architect session with nil config")
	}
}

func TestServerStoreAccess(t *testing.T) {
	server, cleanup := setupTestServer(t, "")
	defer cleanup()

	// Verify we can access the store
	store := server.Store()
	if store == nil {
		t.Fatal("store should not be nil")
	}

	// Create a ticket to verify store works
	ticket, err := store.Create("Test Ticket", "body")
	if err != nil {
		t.Fatalf("failed to create ticket: %v", err)
	}

	if ticket.ID == "" {
		t.Error("ticket ID should not be empty")
	}
}
