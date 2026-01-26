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
		TicketsDir: tmpDir,
	}

	if ticketID != "" {
		cfg.TicketID = ticketID
		cfg.DaemonURL = "http://localhost:4200" // ticket sessions require daemon URL
		cfg.TicketsDir = ""                     // ticket sessions don't use local store
		cfg.ProjectPath = tmpDir                // needed for SDK client project header
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
	if server.sdkClient == nil {
		t.Error("ticket session should have SDK client")
	}
	if server.store != nil {
		t.Error("ticket session should not have local store")
	}
}

func TestNewServerTicketRequiresDaemonURL(t *testing.T) {
	_, err := NewServer(&Config{
		TicketID:   "test-ticket-123",
		TicketsDir: "/tmp/test",
	})
	if err == nil {
		t.Fatal("expected error when DaemonURL is missing for ticket session")
	}
	expected := "ticket sessions require CORTEX_DAEMON_URL to be set"
	if err.Error() != expected {
		t.Errorf("error = %q, want %q", err.Error(), expected)
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
	// With nil config (no ProjectPath or TicketsDir), NewServer should fail
	_, err := NewServer(nil)
	if err == nil {
		t.Fatal("expected error with nil config (no ProjectPath or TicketsDir)")
	}
	expectedMsg := "MCP server requires CORTEX_PROJECT_PATH or CORTEX_TICKETS_DIR to be set"
	if err.Error() != expectedMsg {
		t.Errorf("error = %q, want %q", err.Error(), expectedMsg)
	}
}

func TestNewServerWithProjectPath(t *testing.T) {
	// Create a temp directory to act as project path
	tmpDir, err := os.MkdirTemp("", "mcp-project-test")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cfg := &Config{
		ProjectPath: tmpDir,
	}

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	// Should default to architect session
	if !server.IsArchitectSession() {
		t.Error("expected architect session with project path")
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
