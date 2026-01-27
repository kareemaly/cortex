package mcp

import (
	"os"
	"testing"
)

func TestNewServerArchitect(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mcp-server-test")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	server, err := NewServer(&Config{
		ProjectPath: tmpDir,
		DaemonURL:   "http://localhost:4200",
	})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

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
	tmpDir, err := os.MkdirTemp("", "mcp-server-test")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	server, err := NewServer(&Config{
		TicketID:    "test-ticket-123",
		DaemonURL:   "http://localhost:4200",
		ProjectPath: tmpDir,
	})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

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
}

func TestNewServerTicketRequiresDaemonURL(t *testing.T) {
	_, err := NewServer(&Config{
		TicketID: "test-ticket-123",
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
	tmpDir, err := os.MkdirTemp("", "mcp-default-test")
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
		t.Error("expected architect session by default")
	}
}

func TestNewServerNilConfig(t *testing.T) {
	// With nil config (no ProjectPath), NewServer should fail
	_, err := NewServer(nil)
	if err == nil {
		t.Fatal("expected error with nil config (no ProjectPath)")
	}
	expectedMsg := "MCP server requires CORTEX_PROJECT_PATH to be set"
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
