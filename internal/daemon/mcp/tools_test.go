package mcp

import (
	"context"
	"log/slog"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/daemon/api"
	"github.com/kareemaly/cortex/internal/session"
	"github.com/kareemaly/cortex/internal/ticket"
	"github.com/kareemaly/cortex/internal/tmux"
)

// testTicketAgentPrompt is a minimal prompt for testing.
const testTicketAgentPrompt = "## Test Instructions"

// setupArchitectWithDaemon creates an MCP server backed by an HTTP test server
// running the daemon API. windowExists controls whether the mock tmux reports
// windows as existing (false simulates orphaned sessions).
// Returns the MCP server, the underlying ticket store (for test data setup), and a cleanup function.
func setupArchitectWithDaemon(t *testing.T, windowExists bool) (*Server, *ticket.Store, *session.Store, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "mcp-spawn-test")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	// Create the prompts directory and default templates for the project
	ticketPromptDir := filepath.Join(tmpDir, ".cortex", "prompts", "ticket", "work")
	if err := os.MkdirAll(ticketPromptDir, 0755); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("create prompts dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ticketPromptDir, "SYSTEM.md"), []byte(testTicketAgentPrompt), 0644); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("create SYSTEM.md: %v", err)
	}

	// Create project config with name (used as tmux session name)
	configPath := filepath.Join(tmpDir, ".cortex", "cortex.yaml")
	if err := os.WriteFile(configPath, []byte("name: test-session\n"), 0644); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("create cortex.yaml: %v", err)
	}

	// Create mock tmux runner for the daemon
	mockRunner := tmux.NewMockRunner()
	if !windowExists {
		mockRunner.SetWindowExists(false)
	}
	tmuxMgr := tmux.NewManagerWithRunner(mockRunner)

	// Create HTTP test server with daemon API
	logger := slog.Default()
	storeManager := api.NewStoreManager(logger, nil)
	sessionManager := api.NewSessionManager(logger)
	deps := &api.Dependencies{
		StoreManager:   storeManager,
		SessionManager: sessionManager,
		TmuxManager:    tmuxMgr,
		Logger:         logger,
		CortexdPath:    "/mock/cortexd",
	}
	router := api.NewRouter(deps, logger)
	ts := httptest.NewServer(router)

	// Create MCP server with DaemonURL pointing to test server
	cfg := &Config{
		ProjectPath: tmpDir,
		DaemonURL:   ts.URL,
	}

	server, err := NewServer(cfg)
	if err != nil {
		ts.Close()
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("create server: %v", err)
	}

	// Get the underlying store from the StoreManager for test data setup
	store, err := storeManager.GetStore(tmpDir)
	if err != nil {
		ts.Close()
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("get store: %v", err)
	}

	// Get session store for test data setup
	sessStore := sessionManager.GetStore(tmpDir)

	cleanup := func() {
		ts.Close()
		_ = os.RemoveAll(tmpDir)
	}

	return server, store, sessStore, cleanup
}

// Architect tool tests

func TestHandleListTickets(t *testing.T) {
	server, store, _, cleanup := setupArchitectWithDaemon(t, true)
	defer cleanup()

	// Create some tickets
	_, _ = store.Create("Ticket 1", "body 1", "", nil, nil, nil)
	_, _ = store.Create("Ticket 2", "body 2", "", nil, nil, nil)

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
	server, store, _, cleanup := setupArchitectWithDaemon(t, true)
	defer cleanup()

	// Create tickets in different statuses
	t1, _ := store.Create("Backlog Ticket", "", "", nil, nil, nil)
	t2, _ := store.Create("Progress Ticket", "", "", nil, nil, nil)
	_ = store.Move(t2.ID, ticket.StatusProgress)

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
	server, store, _, cleanup := setupArchitectWithDaemon(t, true)
	defer cleanup()

	// Create tickets
	_, _ = store.Create("Fix login bug", "Authentication issue", "", nil, nil, nil)
	_, _ = store.Create("Add feature", "New feature", "", nil, nil, nil)

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
	server, store, _, cleanup := setupArchitectWithDaemon(t, true)
	defer cleanup()

	// Create tickets
	t1, _ := store.Create("Fix login bug", "Authentication issue", "", nil, nil, nil)
	t2, _ := store.Create("Fix login feature", "Another login issue", "", nil, nil, nil)
	_ = store.Move(t2.ID, ticket.StatusProgress)

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
	server, store, _, cleanup := setupArchitectWithDaemon(t, true)
	defer cleanup()

	// Create tickets
	_, _ = store.Create("Ticket 1", "body 1", "", nil, nil, nil)
	_, _ = store.Create("Ticket 2", "body 2", "", nil, nil, nil)

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
	server, _, _, cleanup := setupArchitectWithDaemon(t, true)
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
	server, _, _, cleanup := setupArchitectWithDaemon(t, true)
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
	server, store, _, cleanup := setupArchitectWithDaemon(t, true)
	defer cleanup()

	created, _ := store.Create("Test Ticket", "body", "", nil, nil, nil)

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
	server, _, _, cleanup := setupArchitectWithDaemon(t, true)
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
	server, _, _, cleanup := setupArchitectWithDaemon(t, true)
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
	server, _, _, cleanup := setupArchitectWithDaemon(t, true)
	defer cleanup()

	_, _, err := server.handleCreateTicket(context.Background(), nil, CreateTicketInput{
		Title: "",
	})

	if err == nil {
		t.Error("expected error for empty title")
	}
}

func TestHandleUpdateTicket(t *testing.T) {
	server, store, _, cleanup := setupArchitectWithDaemon(t, true)
	defer cleanup()

	created, _ := store.Create("Original", "body", "", nil, nil, nil)
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
	server, store, _, cleanup := setupArchitectWithDaemon(t, true)
	defer cleanup()

	created, _ := store.Create("To Delete", "", "", nil, nil, nil)

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
	_, _, err = store.Get(created.ID)
	if !ticket.IsNotFound(err) {
		t.Error("ticket should be deleted")
	}
}

func TestHandleMoveTicket(t *testing.T) {
	server, store, _, cleanup := setupArchitectWithDaemon(t, true)
	defer cleanup()

	created, _ := store.Create("Test", "", "", nil, nil, nil)

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
	server, store, _, cleanup := setupArchitectWithDaemon(t, true)
	defer cleanup()

	created, _ := store.Create("Test", "", "", nil, nil, nil)

	_, _, err := server.handleMoveTicket(context.Background(), nil, MoveTicketInput{
		ID:     created.ID,
		Status: "invalid",
	})

	if err == nil {
		t.Error("expected error for invalid status")
	}
}

func TestHandleSpawnSession(t *testing.T) {
	server, store, _, cleanup := setupArchitectWithDaemon(t, true)
	defer cleanup()

	// Create a ticket first
	created, err := store.Create("Test Spawn Session", "Test body", "", nil, nil, nil)
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

	// Should have a tmux window name
	if output.TmuxWindow == "" {
		t.Error("expected tmux window name")
	}
}

func TestHandleSpawnSessionEmptyTicketID(t *testing.T) {
	server, _, _, cleanup := setupArchitectWithDaemon(t, true)
	defer cleanup()

	_, _, err := server.handleSpawnSession(context.Background(), nil, SpawnSessionInput{
		TicketID: "",
	})
	if err == nil {
		t.Error("expected error for empty ticket_id")
	}
}

func TestHandleSpawnSessionTicketNotFound(t *testing.T) {
	server, _, _, cleanup := setupArchitectWithDaemon(t, true)
	defer cleanup()

	_, _, err := server.handleSpawnSession(context.Background(), nil, SpawnSessionInput{
		TicketID: "nonexistent-ticket",
	})
	if err == nil {
		t.Error("expected error for nonexistent ticket")
	}
}

func TestHandleSpawnSessionActiveSession(t *testing.T) {
	server, store, sessStore, cleanup := setupArchitectWithDaemon(t, true)
	defer cleanup()

	// Create a ticket with an active session
	created, err := store.Create("Test Active Session", "Test body", "", nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create ticket: %v", err)
	}
	_, _, err = sessStore.Create(created.ID, "claude", "window", nil, nil)
	if err != nil {
		t.Fatalf("failed to set session: %v", err)
	}

	_, output, err := server.handleSpawnSession(context.Background(), nil, SpawnSessionInput{
		TicketID: created.ID,
	})

	// Should return STATE_CONFLICT error for active session
	if err == nil {
		t.Fatal("expected STATE_CONFLICT error for ticket with active session")
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
}

func TestHandleSpawnSessionAutoMovesToProgress(t *testing.T) {
	server, store, _, cleanup := setupArchitectWithDaemon(t, true)
	defer cleanup()

	// Create a ticket in backlog
	created, err := store.Create("Test Auto Move", "Test body", "", nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create ticket: %v", err)
	}

	// Verify ticket is in backlog before spawn
	_, statusBefore, err := store.Get(created.ID)
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

	// Verify ticket auto-moved to progress
	_, statusAfter, err := store.Get(created.ID)
	if err != nil {
		t.Fatalf("failed to get ticket after spawn: %v", err)
	}
	if statusAfter != ticket.StatusProgress {
		t.Errorf("ticket should be in progress after spawn, got %v", statusAfter)
	}
}

// Ticket tool tests

// setupTicketSession creates a ticket session backed by a real HTTP test server.
// The MCP ticket handlers will route through the daemon API via the SDK client.
func setupTicketSession(t *testing.T) (*Server, string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "mcp-ticket-test")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	// Create a store and ticket via the daemon API infrastructure
	store, err := ticket.NewStore(filepath.Join(tmpDir, ".cortex", "tickets"), nil, "")
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("create store: %v", err)
	}

	// Create a ticket
	tk, err := store.Create("Test Ticket", "body", "", nil, nil, nil)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("create ticket: %v", err)
	}

	// Set a session on the ticket
	sessionsPath := filepath.Join(tmpDir, ".cortex", "sessions.json")
	localSessStore := session.NewStore(sessionsPath)
	_, _, err = localSessStore.Create(tk.ID, "claude", "window", nil, nil)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("set session: %v", err)
	}

	// Start an HTTP test server with the daemon API
	logger := slog.Default()
	storeManager := api.NewStoreManager(logger, nil)
	sessionManager := api.NewSessionManager(logger)
	deps := &api.Dependencies{
		StoreManager:   storeManager,
		SessionManager: sessionManager,
		Logger:         logger,
		CortexdPath:    "/mock/cortexd",
	}
	router := api.NewRouter(deps, logger)
	ts := httptest.NewServer(router)

	// Create SDK client pointing to test server
	sdkClient := sdk.NewClient(ts.URL, tmpDir)

	// Create MCP server with SDK client (ticket session)
	cfg := &Config{
		TicketID:    tk.ID,
		DaemonURL:   ts.URL,
		ProjectPath: tmpDir,
	}

	mcpServer, err := NewServer(cfg)
	if err != nil {
		ts.Close()
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("create server: %v", err)
	}

	// Override the SDK client to use the test server's client
	mcpServer.sdkClient = sdkClient

	cleanup := func() {
		ts.Close()
		_ = os.RemoveAll(tmpDir)
	}

	return mcpServer, tk.ID, cleanup
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

func TestHandleAddComment(t *testing.T) {
	server, _, cleanup := setupTicketSession(t)
	defer cleanup()

	_, output, err := server.handleAddComment(context.Background(), nil, AddCommentInput{
		Content: "Decided to use new API",
	})
	if err != nil {
		t.Fatalf("handleAddComment failed: %v", err)
	}

	if !output.Success {
		t.Error("add comment should succeed")
	}
	if output.Comment.ID == "" {
		t.Error("comment ID should not be empty")
	}
	if output.Comment.Type != "comment" {
		t.Errorf("comment type = %q, want 'comment'", output.Comment.Type)
	}
	if output.Comment.Content != "Decided to use new API" {
		t.Errorf("comment content = %q, want 'Decided to use new API'", output.Comment.Content)
	}
}

func TestHandleAddBlocker(t *testing.T) {
	server, _, cleanup := setupTicketSession(t)
	defer cleanup()

	_, output, err := server.handleAddBlocker(context.Background(), nil, AddBlockerInput{
		Content: "Blocked by dependency issue",
	})
	if err != nil {
		t.Fatalf("handleAddBlocker failed: %v", err)
	}

	if !output.Success {
		t.Error("add blocker should succeed")
	}
	if output.Comment.Type != "blocker" {
		t.Errorf("comment type = %q, want 'blocker'", output.Comment.Type)
	}
}

// SpawnSession state/mode matrix tests

func TestHandleSpawnSession_StateNormal_ModeNormal(t *testing.T) {
	server, store, _, cleanup := setupArchitectWithDaemon(t, true)
	defer cleanup()

	created, _ := store.Create("Test Ticket", "body", "", nil, nil, nil)

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
}

func TestHandleSpawnSession_StateNormal_ModeResume(t *testing.T) {
	server, store, _, cleanup := setupArchitectWithDaemon(t, true)
	defer cleanup()

	created, _ := store.Create("Test Ticket", "body", "", nil, nil, nil)

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
	server, store, _, cleanup := setupArchitectWithDaemon(t, true)
	defer cleanup()

	created, _ := store.Create("Test Ticket", "body", "", nil, nil, nil)

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
			server, store, sessStore, cleanup := setupArchitectWithDaemon(t, true)
			defer cleanup()

			// Create ticket with active session (window exists because mock defaults to true)
			created, _ := store.Create("Test Ticket", "body", "", nil, nil, nil)
			_, _, _ = sessStore.Create(created.ID, "claude", "window", nil, nil)

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
	server, store, sessStore, cleanup := setupArchitectWithDaemon(t, false)
	defer cleanup()

	created, _ := store.Create("Test Ticket", "body", "", nil, nil, nil)
	_, _, _ = sessStore.Create(created.ID, "claude", "window", nil, nil)

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
	server, store, sessStore, cleanup := setupArchitectWithDaemon(t, false)
	defer cleanup()

	created, _ := store.Create("Test Ticket", "body", "", nil, nil, nil)
	_, _, _ = sessStore.Create(created.ID, "claude", "window", nil, nil)

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
}

func TestHandleSpawnSession_StateOrphaned_ModeFresh(t *testing.T) {
	server, store, sessStore, cleanup := setupArchitectWithDaemon(t, false)
	defer cleanup()

	created, _ := store.Create("Test Ticket", "body", "", nil, nil, nil)
	_, _, _ = sessStore.Create(created.ID, "claude", "window", nil, nil)

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
}

// StateEnded tests removed: sessions are ephemeral (deleted on end),
// so Create+End = no session = StateNormal (already tested above).

func TestHandleSpawnSession_InvalidMode(t *testing.T) {
	server, store, _, cleanup := setupArchitectWithDaemon(t, true)
	defer cleanup()

	created, _ := store.Create("Test Ticket", "body", "", nil, nil, nil)

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
	server, store, _, cleanup := setupArchitectWithDaemon(t, true)
	defer cleanup()

	created, _ := store.Create("Test Ticket", "body", "", nil, nil, nil)

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
}

func TestHandleConcludeSession(t *testing.T) {
	server, ticketID, cleanup := setupTicketSession(t)
	defer cleanup()

	_, output, err := server.handleConcludeSession(context.Background(), nil, ConcludeSessionInput{
		Content: "Work completed successfully",
	})
	if err != nil {
		t.Fatalf("handleConcludeSession failed: %v", err)
	}

	if !output.Success {
		t.Errorf("expected success, got: %+v", output)
	}
	if output.TicketID != ticketID {
		t.Errorf("ticket ID = %q, want %q", output.TicketID, ticketID)
	}
}

func TestHandleAddCommentMissingContent(t *testing.T) {
	server, _, cleanup := setupTicketSession(t)
	defer cleanup()

	_, _, err := server.handleAddComment(context.Background(), nil, AddCommentInput{
		Content: "",
	})
	if err == nil {
		t.Error("expected error for missing content")
	}
	toolErr, ok := err.(*ToolError)
	if !ok {
		t.Fatalf("expected ToolError, got %T", err)
	}
	if toolErr.Code != ErrorCodeValidation {
		t.Errorf("error code = %q, want %q", toolErr.Code, ErrorCodeValidation)
	}
}

func TestHandleRequestReview(t *testing.T) {
	server, _, cleanup := setupTicketSession(t)
	defer cleanup()

	_, output, err := server.handleRequestReview(context.Background(), nil, RequestReviewInput{
		RepoPath: ".",
		Content:  "Please review the implementation",
	})
	if err != nil {
		t.Fatalf("handleRequestReview failed: %v", err)
	}

	if !output.Success {
		t.Error("request review should succeed")
	}
	if output.Comment.ID == "" {
		t.Error("comment ID should not be empty")
	}
	if output.Comment.Type != "review_requested" {
		t.Errorf("comment type = %q, want 'review_requested'", output.Comment.Type)
	}
}

func TestHandleRequestReviewValidation(t *testing.T) {
	server, _, cleanup := setupTicketSession(t)
	defer cleanup()

	// Missing repo_path
	_, _, err := server.handleRequestReview(context.Background(), nil, RequestReviewInput{
		RepoPath: "",
		Content:  "Some content",
	})
	if err == nil {
		t.Error("expected error for missing repo_path")
	}
	toolErr, ok := err.(*ToolError)
	if !ok {
		t.Fatalf("expected ToolError, got %T", err)
	}
	if toolErr.Code != ErrorCodeValidation {
		t.Errorf("error code = %q, want %q", toolErr.Code, ErrorCodeValidation)
	}

	// Missing content
	_, _, err = server.handleRequestReview(context.Background(), nil, RequestReviewInput{
		RepoPath: ".",
		Content:  "",
	})
	if err == nil {
		t.Error("expected error for missing content")
	}
	toolErr, ok = err.(*ToolError)
	if !ok {
		t.Fatalf("expected ToolError, got %T", err)
	}
	if toolErr.Code != ErrorCodeValidation {
		t.Errorf("error code = %q, want %q", toolErr.Code, ErrorCodeValidation)
	}
}
