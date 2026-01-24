//go:build integration

package mcp

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// testEnv holds the test environment configuration.
type testEnv struct {
	ticketsDir string
	binPath    string
	cleanup    func()
}

// mcpTestClient wraps an MCP client session for testing.
type mcpTestClient struct {
	session *mcp.ClientSession
	cmd     *exec.Cmd
	cancel  context.CancelFunc
}

// skipIfCI skips the test if running in CI environment.
func skipIfCI(t *testing.T) {
	t.Helper()
	if os.Getenv("CI") != "" {
		t.Skip("skipping integration test in CI environment")
	}
}

// projectRoot finds the project root directory.
func projectRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get current file path")
	}

	// Navigate up from internal/daemon/mcp/file.go to project root
	// file.go -> internal/daemon/mcp -> internal/daemon -> internal -> project root
	dir := filepath.Dir(filename) // internal/daemon/mcp
	for i := 0; i < 3; i++ {
		dir = filepath.Dir(dir)
	}

	// Verify we're at the project root by checking for go.mod
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		t.Fatalf("could not find project root from %s (tried %s)", filename, dir)
	}

	return dir
}

// setupTestEnv creates a temporary test environment with the necessary directories.
func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	// Create temp directory for tickets
	ticketsDir, err := os.MkdirTemp("", "mcp-integration-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Create status subdirectories
	for _, status := range []string{"backlog", "progress", "done"} {
		if err := os.MkdirAll(filepath.Join(ticketsDir, status), 0755); err != nil {
			os.RemoveAll(ticketsDir)
			t.Fatalf("failed to create %s dir: %v", status, err)
		}
	}

	// Find cortexd binary
	root := projectRoot(t)
	binPath := filepath.Join(root, "bin", "cortexd")

	// Check if binary exists
	if _, err := os.Stat(binPath); err != nil {
		os.RemoveAll(ticketsDir)
		t.Fatalf("cortexd binary not found at %s. Run 'make build' first.", binPath)
	}

	cleanup := func() {
		os.RemoveAll(ticketsDir)
	}

	return &testEnv{
		ticketsDir: ticketsDir,
		binPath:    binPath,
		cleanup:    cleanup,
	}
}

// newMCPTestClient spawns cortexd mcp and connects via MCP SDK.
func newMCPTestClient(t *testing.T, env *testEnv) *mcpTestClient {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "v1.0.0",
	}, nil)

	// Create command transport
	cmd := exec.Command(env.binPath, "mcp")
	cmd.Env = append(os.Environ(),
		"CORTEX_TICKETS_DIR="+env.ticketsDir,
	)

	transport := &mcp.CommandTransport{Command: cmd}

	// Connect to the server
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		cancel()
		t.Fatalf("failed to connect to MCP server: %v", err)
	}

	return &mcpTestClient{
		session: session,
		cmd:     cmd,
		cancel:  cancel,
	}
}

// Close closes the MCP client connection.
func (c *mcpTestClient) Close() {
	if c.session != nil {
		c.session.Close()
	}
	c.cancel()
}

// callTool calls an MCP tool and returns the result.
func (c *mcpTestClient) callTool(ctx context.Context, name string, args map[string]any) (*mcp.CallToolResult, error) {
	params := &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	}
	return c.session.CallTool(ctx, params)
}

// parseToolOutput parses the tool result content into the specified type.
func parseToolOutput[T any](t *testing.T, result *mcp.CallToolResult) T {
	t.Helper()

	if len(result.Content) == 0 {
		t.Fatal("no content in tool result")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}

	var output T
	if err := json.Unmarshal([]byte(textContent.Text), &output); err != nil {
		t.Fatalf("failed to unmarshal tool output: %v\nraw: %s", err, textContent.Text)
	}

	return output
}

// Integration tests for architect tools

func TestIntegration_ListTickets_MissingStatus(t *testing.T) {
	skipIfCI(t)

	env := setupTestEnv(t)
	defer env.cleanup()

	client := newMCPTestClient(t, env)
	defer client.Close()

	ctx := context.Background()
	result, err := client.callTool(ctx, "listTickets", nil)

	// The MCP SDK validates the schema at the transport level, returning an error
	// when required fields are missing. This is the expected behavior.
	if err == nil && (result == nil || !result.IsError) {
		t.Error("expected error for missing status")
	}
	// If we got an error, verify it's about the missing status field
	if err != nil && !strings.Contains(err.Error(), "status") {
		t.Errorf("expected error about status, got: %v", err)
	}
}

func TestIntegration_ListTickets_InvalidStatus(t *testing.T) {
	skipIfCI(t)

	env := setupTestEnv(t)
	defer env.cleanup()

	client := newMCPTestClient(t, env)
	defer client.Close()

	ctx := context.Background()
	result, err := client.callTool(ctx, "listTickets", map[string]any{
		"status": "invalid",
	})
	if err != nil {
		t.Fatalf("callTool failed: %v", err)
	}

	// Should have an error for invalid status
	if !result.IsError {
		t.Error("expected error for invalid status")
	}
}

func TestIntegration_ListTickets_Empty(t *testing.T) {
	skipIfCI(t)

	env := setupTestEnv(t)
	defer env.cleanup()

	client := newMCPTestClient(t, env)
	defer client.Close()

	ctx := context.Background()
	result, err := client.callTool(ctx, "listTickets", map[string]any{
		"status": "backlog",
	})
	if err != nil {
		t.Fatalf("listTickets failed: %v", err)
	}

	output := parseToolOutput[ListTicketsOutput](t, result)
	if output.Total != 0 {
		t.Errorf("expected 0 tickets, got %d", output.Total)
	}
	if len(output.Tickets) != 0 {
		t.Errorf("expected empty tickets slice, got %d", len(output.Tickets))
	}
}

func TestIntegration_CreateTicket(t *testing.T) {
	skipIfCI(t)

	env := setupTestEnv(t)
	defer env.cleanup()

	client := newMCPTestClient(t, env)
	defer client.Close()

	ctx := context.Background()
	result, err := client.callTool(ctx, "createTicket", map[string]any{
		"title": "Test Integration Ticket",
		"body":  "This is a test body",
	})
	if err != nil {
		t.Fatalf("createTicket failed: %v", err)
	}

	output := parseToolOutput[CreateTicketOutput](t, result)
	if output.Ticket.ID == "" {
		t.Error("expected ticket ID to be set")
	}
	if output.Ticket.Title != "Test Integration Ticket" {
		t.Errorf("expected title 'Test Integration Ticket', got %q", output.Ticket.Title)
	}
	if output.Ticket.Body != "This is a test body" {
		t.Errorf("expected body 'This is a test body', got %q", output.Ticket.Body)
	}
	if output.Ticket.Status != "backlog" {
		t.Errorf("expected status 'backlog', got %q", output.Ticket.Status)
	}
}

func TestIntegration_CreateTicket_EmptyTitle(t *testing.T) {
	skipIfCI(t)

	env := setupTestEnv(t)
	defer env.cleanup()

	client := newMCPTestClient(t, env)
	defer client.Close()

	ctx := context.Background()
	result, err := client.callTool(ctx, "createTicket", map[string]any{
		"title": "",
	})
	if err != nil {
		t.Fatalf("callTool failed: %v", err)
	}

	// Should have an error in the result
	if !result.IsError {
		t.Error("expected error for empty title")
	}
}

func TestIntegration_ListTickets_WithTickets(t *testing.T) {
	skipIfCI(t)

	env := setupTestEnv(t)
	defer env.cleanup()

	client := newMCPTestClient(t, env)
	defer client.Close()

	ctx := context.Background()

	// Create some tickets
	_, err := client.callTool(ctx, "createTicket", map[string]any{
		"title": "Ticket 1",
	})
	if err != nil {
		t.Fatalf("createTicket failed: %v", err)
	}

	_, err = client.callTool(ctx, "createTicket", map[string]any{
		"title": "Ticket 2",
	})
	if err != nil {
		t.Fatalf("createTicket failed: %v", err)
	}

	// List backlog tickets (status is required)
	result, err := client.callTool(ctx, "listTickets", map[string]any{
		"status": "backlog",
	})
	if err != nil {
		t.Fatalf("listTickets failed: %v", err)
	}

	output := parseToolOutput[ListTicketsOutput](t, result)
	if output.Total != 2 {
		t.Errorf("expected 2 tickets, got %d", output.Total)
	}
}

func TestIntegration_ListTickets_FilterByStatus(t *testing.T) {
	skipIfCI(t)

	env := setupTestEnv(t)
	defer env.cleanup()

	client := newMCPTestClient(t, env)
	defer client.Close()

	ctx := context.Background()

	// Create a ticket
	createResult, err := client.callTool(ctx, "createTicket", map[string]any{
		"title": "Backlog Ticket",
	})
	if err != nil {
		t.Fatalf("createTicket failed: %v", err)
	}
	ticket1 := parseToolOutput[CreateTicketOutput](t, createResult)

	// Create another and move to progress
	createResult2, err := client.callTool(ctx, "createTicket", map[string]any{
		"title": "Progress Ticket",
	})
	if err != nil {
		t.Fatalf("createTicket failed: %v", err)
	}
	ticket2 := parseToolOutput[CreateTicketOutput](t, createResult2)

	_, err = client.callTool(ctx, "moveTicket", map[string]any{
		"id":     ticket2.Ticket.ID,
		"status": "progress",
	})
	if err != nil {
		t.Fatalf("moveTicket failed: %v", err)
	}

	// List only backlog tickets
	result, err := client.callTool(ctx, "listTickets", map[string]any{
		"status": "backlog",
	})
	if err != nil {
		t.Fatalf("listTickets failed: %v", err)
	}

	output := parseToolOutput[ListTicketsOutput](t, result)
	if output.Total != 1 {
		t.Errorf("expected 1 backlog ticket, got %d", output.Total)
	}
	if output.Tickets[0].ID != ticket1.Ticket.ID {
		t.Errorf("expected ticket ID %s, got %s", ticket1.Ticket.ID, output.Tickets[0].ID)
	}
}

func TestIntegration_ReadTicket(t *testing.T) {
	skipIfCI(t)

	env := setupTestEnv(t)
	defer env.cleanup()

	client := newMCPTestClient(t, env)
	defer client.Close()

	ctx := context.Background()

	// Create a ticket
	createResult, err := client.callTool(ctx, "createTicket", map[string]any{
		"title": "Test Read Ticket",
		"body":  "Test body content",
	})
	if err != nil {
		t.Fatalf("createTicket failed: %v", err)
	}
	created := parseToolOutput[CreateTicketOutput](t, createResult)

	// Read the ticket
	result, err := client.callTool(ctx, "readTicket", map[string]any{
		"id": created.Ticket.ID,
	})
	if err != nil {
		t.Fatalf("readTicket failed: %v", err)
	}

	output := parseToolOutput[ReadTicketOutput](t, result)
	if output.Ticket.ID != created.Ticket.ID {
		t.Errorf("expected ID %s, got %s", created.Ticket.ID, output.Ticket.ID)
	}
	if output.Ticket.Title != "Test Read Ticket" {
		t.Errorf("expected title 'Test Read Ticket', got %q", output.Ticket.Title)
	}
	if output.Ticket.Body != "Test body content" {
		t.Errorf("expected body 'Test body content', got %q", output.Ticket.Body)
	}
}

func TestIntegration_ReadTicket_NotFound(t *testing.T) {
	skipIfCI(t)

	env := setupTestEnv(t)
	defer env.cleanup()

	client := newMCPTestClient(t, env)
	defer client.Close()

	ctx := context.Background()
	result, err := client.callTool(ctx, "readTicket", map[string]any{
		"id": "nonexistent-ticket-id",
	})
	if err != nil {
		t.Fatalf("callTool failed: %v", err)
	}

	// Should have an error
	if !result.IsError {
		t.Error("expected error for nonexistent ticket")
	}
}

func TestIntegration_UpdateTicket(t *testing.T) {
	skipIfCI(t)

	env := setupTestEnv(t)
	defer env.cleanup()

	client := newMCPTestClient(t, env)
	defer client.Close()

	ctx := context.Background()

	// Create a ticket
	createResult, err := client.callTool(ctx, "createTicket", map[string]any{
		"title": "Original Title",
		"body":  "Original body",
	})
	if err != nil {
		t.Fatalf("createTicket failed: %v", err)
	}
	created := parseToolOutput[CreateTicketOutput](t, createResult)

	// Update the ticket title
	result, err := client.callTool(ctx, "updateTicket", map[string]any{
		"id":    created.Ticket.ID,
		"title": "Updated Title",
	})
	if err != nil {
		t.Fatalf("updateTicket failed: %v", err)
	}

	output := parseToolOutput[UpdateTicketOutput](t, result)
	if output.Ticket.Title != "Updated Title" {
		t.Errorf("expected title 'Updated Title', got %q", output.Ticket.Title)
	}
	// Body should remain unchanged
	if output.Ticket.Body != "Original body" {
		t.Errorf("expected body 'Original body', got %q", output.Ticket.Body)
	}
}

func TestIntegration_UpdateTicket_PartialUpdate(t *testing.T) {
	skipIfCI(t)

	env := setupTestEnv(t)
	defer env.cleanup()

	client := newMCPTestClient(t, env)
	defer client.Close()

	ctx := context.Background()

	// Create a ticket
	createResult, err := client.callTool(ctx, "createTicket", map[string]any{
		"title": "Original Title",
		"body":  "Original body",
	})
	if err != nil {
		t.Fatalf("createTicket failed: %v", err)
	}
	created := parseToolOutput[CreateTicketOutput](t, createResult)

	// Update only the body
	result, err := client.callTool(ctx, "updateTicket", map[string]any{
		"id":   created.Ticket.ID,
		"body": "New body content",
	})
	if err != nil {
		t.Fatalf("updateTicket failed: %v", err)
	}

	output := parseToolOutput[UpdateTicketOutput](t, result)
	if output.Ticket.Title != "Original Title" {
		t.Errorf("expected title 'Original Title', got %q", output.Ticket.Title)
	}
	if output.Ticket.Body != "New body content" {
		t.Errorf("expected body 'New body content', got %q", output.Ticket.Body)
	}
}

func TestIntegration_DeleteTicket(t *testing.T) {
	skipIfCI(t)

	env := setupTestEnv(t)
	defer env.cleanup()

	client := newMCPTestClient(t, env)
	defer client.Close()

	ctx := context.Background()

	// Create a ticket
	createResult, err := client.callTool(ctx, "createTicket", map[string]any{
		"title": "To Be Deleted",
	})
	if err != nil {
		t.Fatalf("createTicket failed: %v", err)
	}
	created := parseToolOutput[CreateTicketOutput](t, createResult)

	// Delete the ticket
	result, err := client.callTool(ctx, "deleteTicket", map[string]any{
		"id": created.Ticket.ID,
	})
	if err != nil {
		t.Fatalf("deleteTicket failed: %v", err)
	}

	output := parseToolOutput[DeleteTicketOutput](t, result)
	if !output.Success {
		t.Error("expected delete to succeed")
	}
	if output.ID != created.Ticket.ID {
		t.Errorf("expected ID %s, got %s", created.Ticket.ID, output.ID)
	}

	// Verify reading the ticket returns not found
	readResult, err := client.callTool(ctx, "readTicket", map[string]any{
		"id": created.Ticket.ID,
	})
	if err != nil {
		t.Fatalf("callTool failed: %v", err)
	}
	if !readResult.IsError {
		t.Error("expected error when reading deleted ticket")
	}
}

func TestIntegration_MoveTicket(t *testing.T) {
	skipIfCI(t)

	env := setupTestEnv(t)
	defer env.cleanup()

	client := newMCPTestClient(t, env)
	defer client.Close()

	ctx := context.Background()

	// Create a ticket in backlog
	createResult, err := client.callTool(ctx, "createTicket", map[string]any{
		"title": "Movable Ticket",
	})
	if err != nil {
		t.Fatalf("createTicket failed: %v", err)
	}
	created := parseToolOutput[CreateTicketOutput](t, createResult)

	// Move to progress
	result, err := client.callTool(ctx, "moveTicket", map[string]any{
		"id":     created.Ticket.ID,
		"status": "progress",
	})
	if err != nil {
		t.Fatalf("moveTicket failed: %v", err)
	}

	output := parseToolOutput[MoveTicketOutput](t, result)
	if !output.Success {
		t.Error("expected move to succeed")
	}
	if output.Status != "progress" {
		t.Errorf("expected status 'progress', got %q", output.Status)
	}

	// Move to done
	result2, err := client.callTool(ctx, "moveTicket", map[string]any{
		"id":     created.Ticket.ID,
		"status": "done",
	})
	if err != nil {
		t.Fatalf("moveTicket failed: %v", err)
	}

	output2 := parseToolOutput[MoveTicketOutput](t, result2)
	if output2.Status != "done" {
		t.Errorf("expected status 'done', got %q", output2.Status)
	}
}

func TestIntegration_MoveTicket_InvalidStatus(t *testing.T) {
	skipIfCI(t)

	env := setupTestEnv(t)
	defer env.cleanup()

	client := newMCPTestClient(t, env)
	defer client.Close()

	ctx := context.Background()

	// Create a ticket
	createResult, err := client.callTool(ctx, "createTicket", map[string]any{
		"title": "Test Ticket",
	})
	if err != nil {
		t.Fatalf("createTicket failed: %v", err)
	}
	created := parseToolOutput[CreateTicketOutput](t, createResult)

	// Try to move to invalid status
	result, err := client.callTool(ctx, "moveTicket", map[string]any{
		"id":     created.Ticket.ID,
		"status": "invalid_status",
	})
	if err != nil {
		t.Fatalf("callTool failed: %v", err)
	}

	if !result.IsError {
		t.Error("expected error for invalid status")
	}
}

func TestIntegration_ListTickets_WithQuery(t *testing.T) {
	skipIfCI(t)

	env := setupTestEnv(t)
	defer env.cleanup()

	client := newMCPTestClient(t, env)
	defer client.Close()

	ctx := context.Background()

	// Create tickets with different content
	_, err := client.callTool(ctx, "createTicket", map[string]any{
		"title": "Fix login authentication bug",
		"body":  "Users cannot login with valid credentials",
	})
	if err != nil {
		t.Fatalf("createTicket failed: %v", err)
	}

	_, err = client.callTool(ctx, "createTicket", map[string]any{
		"title": "Add new feature",
		"body":  "Implement dashboard widget",
	})
	if err != nil {
		t.Fatalf("createTicket failed: %v", err)
	}

	// Search for "login" in backlog (status is required)
	result, err := client.callTool(ctx, "listTickets", map[string]any{
		"status": "backlog",
		"query":  "login",
	})
	if err != nil {
		t.Fatalf("listTickets failed: %v", err)
	}

	output := parseToolOutput[ListTicketsOutput](t, result)
	if output.Total != 1 {
		t.Errorf("expected 1 match for 'login', got %d", output.Total)
	}
	if !strings.Contains(output.Tickets[0].Title, "login") {
		t.Errorf("expected ticket with 'login' in title, got %q", output.Tickets[0].Title)
	}
}

func TestIntegration_ListTickets_WithStatusAndQuery(t *testing.T) {
	skipIfCI(t)

	env := setupTestEnv(t)
	defer env.cleanup()

	client := newMCPTestClient(t, env)
	defer client.Close()

	ctx := context.Background()

	// Create ticket in backlog
	_, err := client.callTool(ctx, "createTicket", map[string]any{
		"title": "Fix login bug in backlog",
		"body":  "Login issue",
	})
	if err != nil {
		t.Fatalf("createTicket failed: %v", err)
	}

	// Create ticket and move to progress
	createResult, err := client.callTool(ctx, "createTicket", map[string]any{
		"title": "Fix login bug in progress",
		"body":  "Another login issue",
	})
	if err != nil {
		t.Fatalf("createTicket failed: %v", err)
	}
	ticket2 := parseToolOutput[CreateTicketOutput](t, createResult)

	_, err = client.callTool(ctx, "moveTicket", map[string]any{
		"id":     ticket2.Ticket.ID,
		"status": "progress",
	})
	if err != nil {
		t.Fatalf("moveTicket failed: %v", err)
	}

	// Search for "login" in backlog only
	result, err := client.callTool(ctx, "listTickets", map[string]any{
		"status": "backlog",
		"query":  "login",
	})
	if err != nil {
		t.Fatalf("listTickets failed: %v", err)
	}

	output := parseToolOutput[ListTicketsOutput](t, result)
	if output.Total != 1 {
		t.Errorf("expected 1 match for 'login' in backlog, got %d", output.Total)
	}
}

// Test full workflow: create -> read -> update -> move -> delete
func TestIntegration_FullWorkflow(t *testing.T) {
	skipIfCI(t)

	env := setupTestEnv(t)
	defer env.cleanup()

	client := newMCPTestClient(t, env)
	defer client.Close()

	ctx := context.Background()

	// 1. Create a ticket
	createResult, err := client.callTool(ctx, "createTicket", map[string]any{
		"title": "Full Workflow Test",
		"body":  "Testing the complete workflow",
	})
	if err != nil {
		t.Fatalf("createTicket failed: %v", err)
	}
	created := parseToolOutput[CreateTicketOutput](t, createResult)
	ticketID := created.Ticket.ID
	t.Logf("Created ticket: %s", ticketID)

	// 2. Read and verify
	readResult, err := client.callTool(ctx, "readTicket", map[string]any{
		"id": ticketID,
	})
	if err != nil {
		t.Fatalf("readTicket failed: %v", err)
	}
	readOutput := parseToolOutput[ReadTicketOutput](t, readResult)
	if readOutput.Ticket.Status != "backlog" {
		t.Errorf("expected status 'backlog', got %q", readOutput.Ticket.Status)
	}

	// 3. Update the ticket
	updateResult, err := client.callTool(ctx, "updateTicket", map[string]any{
		"id":    ticketID,
		"title": "Full Workflow Test - Updated",
		"body":  "Updated body content",
	})
	if err != nil {
		t.Fatalf("updateTicket failed: %v", err)
	}
	updateOutput := parseToolOutput[UpdateTicketOutput](t, updateResult)
	if updateOutput.Ticket.Title != "Full Workflow Test - Updated" {
		t.Errorf("expected updated title, got %q", updateOutput.Ticket.Title)
	}

	// 4. Move to progress
	moveResult, err := client.callTool(ctx, "moveTicket", map[string]any{
		"id":     ticketID,
		"status": "progress",
	})
	if err != nil {
		t.Fatalf("moveTicket failed: %v", err)
	}
	moveOutput := parseToolOutput[MoveTicketOutput](t, moveResult)
	if moveOutput.Status != "progress" {
		t.Errorf("expected status 'progress', got %q", moveOutput.Status)
	}

	// 5. List and verify only in progress
	listResult, err := client.callTool(ctx, "listTickets", map[string]any{
		"status": "progress",
	})
	if err != nil {
		t.Fatalf("listTickets failed: %v", err)
	}
	listOutput := parseToolOutput[ListTicketsOutput](t, listResult)
	if listOutput.Total != 1 {
		t.Errorf("expected 1 ticket in progress, got %d", listOutput.Total)
	}

	// 6. Move to done
	_, err = client.callTool(ctx, "moveTicket", map[string]any{
		"id":     ticketID,
		"status": "done",
	})
	if err != nil {
		t.Fatalf("moveTicket failed: %v", err)
	}

	// 7. Delete the ticket
	deleteResult, err := client.callTool(ctx, "deleteTicket", map[string]any{
		"id": ticketID,
	})
	if err != nil {
		t.Fatalf("deleteTicket failed: %v", err)
	}
	deleteOutput := parseToolOutput[DeleteTicketOutput](t, deleteResult)
	if !deleteOutput.Success {
		t.Error("expected delete to succeed")
	}

	// 8. Verify ticket is gone
	finalReadResult, err := client.callTool(ctx, "readTicket", map[string]any{
		"id": ticketID,
	})
	if err != nil {
		t.Fatalf("callTool failed: %v", err)
	}
	if !finalReadResult.IsError {
		t.Error("expected error when reading deleted ticket")
	}

	t.Log("Full workflow completed successfully")
}
