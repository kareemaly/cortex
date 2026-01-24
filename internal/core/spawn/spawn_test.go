package spawn

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/kareemaly/cortex/internal/ticket"
)

// mockStore implements StoreInterface for testing.
type mockStore struct {
	tickets         map[string]*ticket.Ticket
	sessions        map[string]*ticket.Session
	setErr          error
	endErr          error
	getErr          error
	lastSetID       string
	endSessionCalls []string // tracks which ticket IDs had EndSession called
}

func newMockStore() *mockStore {
	return &mockStore{
		tickets:  make(map[string]*ticket.Ticket),
		sessions: make(map[string]*ticket.Session),
	}
}

func (m *mockStore) Get(id string) (*ticket.Ticket, ticket.Status, error) {
	if m.getErr != nil {
		return nil, "", m.getErr
	}
	t, ok := m.tickets[id]
	if !ok {
		return nil, "", errors.New("ticket not found")
	}
	return t, ticket.StatusBacklog, nil
}

func (m *mockStore) SetSession(ticketID, agent, tmuxWindow string, worktreePath, featureBranch *string) (*ticket.Session, error) {
	if m.setErr != nil {
		return nil, m.setErr
	}
	m.lastSetID = ticketID
	session := &ticket.Session{
		ID:            "session-123",
		StartedAt:     time.Now(),
		Agent:         agent,
		TmuxWindow:    tmuxWindow,
		WorktreePath:  worktreePath,
		FeatureBranch: featureBranch,
	}
	m.sessions[ticketID] = session
	if t, ok := m.tickets[ticketID]; ok {
		t.Session = session
	}
	return session, nil
}

func (m *mockStore) EndSession(ticketID string) error {
	if m.endErr != nil {
		return m.endErr
	}
	m.endSessionCalls = append(m.endSessionCalls, ticketID)
	if s, ok := m.sessions[ticketID]; ok {
		now := time.Now()
		s.EndedAt = &now
	}
	return nil
}

// mockTmuxManager implements TmuxManagerInterface for testing.
type mockTmuxManager struct {
	windows          map[string]bool // window existence by name
	spawnErr         error
	windowExists     bool
	spawnCalls       int
	lastCommand      string
	lastCompanionCmd string
	lastWorkingDir   string
}

func newMockTmuxManager() *mockTmuxManager {
	return &mockTmuxManager{
		windows:      make(map[string]bool),
		windowExists: false,
	}
}

func (m *mockTmuxManager) WindowExists(session, windowName string) (bool, error) {
	return m.windowExists, nil
}

func (m *mockTmuxManager) SpawnAgent(session, windowName, agentCommand, companionCommand, workingDir string) (int, error) {
	if m.spawnErr != nil {
		return 0, m.spawnErr
	}
	m.spawnCalls++
	m.lastCommand = agentCommand
	m.lastCompanionCmd = companionCommand
	m.lastWorkingDir = workingDir
	m.windows[windowName] = true
	return 1, nil
}

func (m *mockTmuxManager) SpawnArchitect(session, windowName, agentCommand, companionCommand, workingDir string) error {
	if m.spawnErr != nil {
		return m.spawnErr
	}
	m.spawnCalls++
	m.lastCommand = agentCommand
	m.lastCompanionCmd = companionCommand
	m.lastWorkingDir = workingDir
	m.windows[windowName] = true
	return nil
}

// Test helpers

func createTestTicket(id, title, body string) *ticket.Ticket {
	return &ticket.Ticket{
		ID:    id,
		Title: title,
		Body:  body,
	}
}

func createTestPromptFile(t *testing.T, projectPath, filename, content string) {
	t.Helper()
	dir := filepath.Join(projectPath, ".cortex", "prompts")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// Tests

func TestSpawn_TicketAgent_Success(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	store := newMockStore()
	tmuxMgr := newMockTmuxManager()

	testTicket := createTestTicket("ticket-1", "Test Ticket", "Test body")
	store.tickets["ticket-1"] = testTicket

	createTestPromptFile(t, tmpDir, "ticket-system.md", "## Test Instructions")

	spawner := NewSpawner(Dependencies{
		Store:        store,
		TmuxManager:  tmuxMgr,
		CortexdPath:  "/usr/bin/cortexd",
		MCPConfigDir: tmpDir,
	})

	// Execute
	result, err := spawner.Spawn(SpawnRequest{
		AgentType:   AgentTypeTicketAgent,
		Agent:       "claude",
		TmuxSession: "test-session",
		ProjectPath: tmpDir,
		TicketsDir:  filepath.Join(tmpDir, "tickets"),
		TicketID:    "ticket-1",
		Ticket:      testTicket,
	})

	// Verify
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Message)
	}
	if result.SessionID != "session-123" {
		t.Errorf("expected session ID 'session-123', got: %s", result.SessionID)
	}
	if result.TmuxWindow != "test-ticket" {
		t.Errorf("expected window 'test-ticket', got: %s", result.TmuxWindow)
	}
	if tmuxMgr.spawnCalls != 1 {
		t.Errorf("expected 1 spawn call, got: %d", tmuxMgr.spawnCalls)
	}
}

func TestSpawn_TicketAgent_AlreadyActive(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	store := newMockStore()
	tmuxMgr := newMockTmuxManager()
	tmuxMgr.windowExists = true // Window exists

	testTicket := createTestTicket("ticket-1", "Test Ticket", "Test body")
	testTicket.Session = &ticket.Session{
		ID:         "existing-session",
		TmuxWindow: "test-ticket",
		StartedAt:  time.Now(),
	}
	store.tickets["ticket-1"] = testTicket

	spawner := NewSpawner(Dependencies{
		Store:       store,
		TmuxManager: tmuxMgr,
		CortexdPath: "/usr/bin/cortexd",
	})

	// Execute
	result, err := spawner.Spawn(SpawnRequest{
		AgentType:   AgentTypeTicketAgent,
		Agent:       "claude",
		TmuxSession: "test-session",
		ProjectPath: tmpDir,
		TicketsDir:  filepath.Join(tmpDir, "tickets"),
		TicketID:    "ticket-1",
		Ticket:      testTicket,
	})

	// Verify
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for already active session")
	}
	if result.Message != "ticket already has an active session" {
		t.Errorf("unexpected message: %s", result.Message)
	}
	if tmuxMgr.spawnCalls != 0 {
		t.Errorf("expected no spawn calls, got: %d", tmuxMgr.spawnCalls)
	}
}

func TestSpawn_CleanupOnFailure(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	store := newMockStore()
	tmuxMgr := newMockTmuxManager()
	tmuxMgr.spawnErr = errors.New("tmux failed")

	testTicket := createTestTicket("ticket-1", "Test Ticket", "Test body")
	store.tickets["ticket-1"] = testTicket

	createTestPromptFile(t, tmpDir, "ticket-system.md", "## Test Instructions")

	spawner := NewSpawner(Dependencies{
		Store:        store,
		TmuxManager:  tmuxMgr,
		CortexdPath:  "/usr/bin/cortexd",
		MCPConfigDir: tmpDir,
	})

	// Execute
	result, err := spawner.Spawn(SpawnRequest{
		AgentType:   AgentTypeTicketAgent,
		Agent:       "claude",
		TmuxSession: "test-session",
		ProjectPath: tmpDir,
		TicketsDir:  filepath.Join(tmpDir, "tickets"),
		TicketID:    "ticket-1",
		Ticket:      testTicket,
	})

	// Verify
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure")
	}

	// Session should have been cleaned up
	if session, ok := store.sessions["ticket-1"]; ok {
		if session.EndedAt == nil {
			t.Error("expected session to be ended after cleanup")
		}
	}
}

func TestResume_Success(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	store := newMockStore()
	tmuxMgr := newMockTmuxManager()

	spawner := NewSpawner(Dependencies{
		Store:        store,
		TmuxManager:  tmuxMgr,
		CortexdPath:  "/usr/bin/cortexd",
		MCPConfigDir: tmpDir,
	})

	// Execute
	result, err := spawner.Resume(ResumeRequest{
		AgentType:   AgentTypeTicketAgent,
		TmuxSession: "test-session",
		ProjectPath: tmpDir,
		TicketsDir:  filepath.Join(tmpDir, "tickets"),
		SessionID:   "session-abc",
		WindowName:  "test-ticket",
		TicketID:    "ticket-1",
	})

	// Verify
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Message)
	}
	if result.TmuxWindow != "test-ticket" {
		t.Errorf("expected window 'test-ticket', got: %s", result.TmuxWindow)
	}
	if tmuxMgr.spawnCalls != 1 {
		t.Errorf("expected 1 spawn call, got: %d", tmuxMgr.spawnCalls)
	}

	// Verify --resume flag in command
	if tmuxMgr.lastCommand == "" {
		t.Error("expected command to be set")
	}
}

func TestResume_NoSessionID(t *testing.T) {
	spawner := NewSpawner(Dependencies{})

	_, err := spawner.Resume(ResumeRequest{
		TmuxSession: "test-session",
		WindowName:  "test-window",
		TicketID:    "ticket-1",
	})

	if err == nil {
		t.Fatal("expected error for missing session ID")
	}
	if !IsConfigError(err) {
		t.Errorf("expected ConfigError, got: %T", err)
	}
}

func TestResume_NoTicketID(t *testing.T) {
	spawner := NewSpawner(Dependencies{})

	_, err := spawner.Resume(ResumeRequest{
		TmuxSession: "test-session",
		WindowName:  "test-window",
		SessionID:   "session-abc",
	})

	if err == nil {
		t.Fatal("expected error for missing ticket ID")
	}
	if !IsConfigError(err) {
		t.Errorf("expected ConfigError, got: %T", err)
	}
}

func TestFresh_ClearsExisting(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	store := newMockStore()
	tmuxMgr := newMockTmuxManager()

	testTicket := createTestTicket("ticket-1", "Test Ticket", "Test body")
	testTicket.Session = &ticket.Session{
		ID:         "old-session",
		TmuxWindow: "old-window",
		StartedAt:  time.Now(),
	}
	store.tickets["ticket-1"] = testTicket
	store.sessions["ticket-1"] = testTicket.Session

	createTestPromptFile(t, tmpDir, "ticket-system.md", "## Test Instructions")

	spawner := NewSpawner(Dependencies{
		Store:        store,
		TmuxManager:  tmuxMgr,
		CortexdPath:  "/usr/bin/cortexd",
		MCPConfigDir: tmpDir,
	})

	// Execute
	result, err := spawner.Fresh(SpawnRequest{
		AgentType:   AgentTypeTicketAgent,
		Agent:       "claude",
		TmuxSession: "test-session",
		ProjectPath: tmpDir,
		TicketsDir:  filepath.Join(tmpDir, "tickets"),
		TicketID:    "ticket-1",
		Ticket:      testTicket,
	})

	// Verify
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Message)
	}

	// EndSession should have been called for the old session
	if !slices.Contains(store.endSessionCalls, "ticket-1") {
		t.Error("expected EndSession to be called for ticket-1")
	}
}

func TestDetectTicketState_Normal(t *testing.T) {
	ticket := createTestTicket("ticket-1", "Test", "Body")
	info, err := DetectTicketState(ticket, "test-session", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.State != StateNormal {
		t.Errorf("expected StateNormal, got: %s", info.State)
	}
}

func TestDetectTicketState_Active(t *testing.T) {
	tmuxMgr := newMockTmuxManager()
	tmuxMgr.windowExists = true

	testTicket := createTestTicket("ticket-1", "Test", "Body")
	testTicket.Session = &ticket.Session{
		ID:         "session-1",
		TmuxWindow: "test-window",
		StartedAt:  time.Now(),
	}

	info, err := DetectTicketState(testTicket, "test-session", tmuxMgr)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.State != StateActive {
		t.Errorf("expected StateActive, got: %s", info.State)
	}
	if !info.WindowExists {
		t.Error("expected WindowExists to be true")
	}
}

func TestDetectTicketState_Orphaned(t *testing.T) {
	tmuxMgr := newMockTmuxManager()
	tmuxMgr.windowExists = false

	testTicket := createTestTicket("ticket-1", "Test", "Body")
	testTicket.Session = &ticket.Session{
		ID:         "session-1",
		TmuxWindow: "test-window",
		StartedAt:  time.Now(),
	}

	info, err := DetectTicketState(testTicket, "test-session", tmuxMgr)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.State != StateOrphaned {
		t.Errorf("expected StateOrphaned, got: %s", info.State)
	}
	if info.WindowExists {
		t.Error("expected WindowExists to be false")
	}
}

func TestDetectTicketState_Ended(t *testing.T) {
	now := time.Now()
	testTicket := createTestTicket("ticket-1", "Test", "Body")
	testTicket.Session = &ticket.Session{
		ID:        "session-1",
		StartedAt: now.Add(-time.Hour),
		EndedAt:   &now,
	}

	info, err := DetectTicketState(testTicket, "test-session", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.State != StateEnded {
		t.Errorf("expected StateEnded, got: %s", info.State)
	}
}

func TestGenerateMCPConfig_WithTicket(t *testing.T) {
	config := GenerateMCPConfig(MCPConfigParams{
		CortexdPath: "/usr/bin/cortexd",
		TicketID:    "ticket-123",
		TicketsDir:  "/path/to/tickets",
		ProjectPath: "/path/to/project",
		TmuxSession: "dev-session",
	})

	server, ok := config.MCPServers["cortex"]
	if !ok {
		t.Fatal("expected 'cortex' server in config")
	}

	if server.Command != "/usr/bin/cortexd" {
		t.Errorf("expected command '/usr/bin/cortexd', got: %s", server.Command)
	}

	expectedArgs := []string{"mcp", "--ticket-id", "ticket-123"}
	if len(server.Args) != len(expectedArgs) {
		t.Fatalf("expected %d args, got: %d", len(expectedArgs), len(server.Args))
	}
	for i, arg := range expectedArgs {
		if server.Args[i] != arg {
			t.Errorf("arg[%d]: expected %s, got: %s", i, arg, server.Args[i])
		}
	}

	if server.Env["CORTEX_TICKETS_DIR"] != "/path/to/tickets" {
		t.Errorf("expected CORTEX_TICKETS_DIR, got: %v", server.Env)
	}
}

func TestBuildClaudeCommand(t *testing.T) {
	tests := []struct {
		name     string
		params   ClaudeCommandParams
		contains []string
	}{
		{
			name: "basic command",
			params: ClaudeCommandParams{
				Prompt:        "Hello world",
				MCPConfigPath: "/path/to/config.json",
			},
			contains: []string{"claude", "'Hello world'", "--mcp-config", "/path/to/config.json"},
		},
		{
			name: "with permission mode",
			params: ClaudeCommandParams{
				Prompt:         "Test prompt",
				MCPConfigPath:  "/config.json",
				PermissionMode: "plan",
			},
			contains: []string{"--permission-mode", "plan"},
		},
		{
			name: "with resume",
			params: ClaudeCommandParams{
				Prompt:   "Test",
				ResumeID: "session-abc",
			},
			contains: []string{"--resume", "session-abc"},
		},
		{
			name: "with single quotes in prompt",
			params: ClaudeCommandParams{
				Prompt: "It's a test",
			},
			contains: []string{"'It'\\''s a test'"},
		},
		{
			name: "with allowed tools",
			params: ClaudeCommandParams{
				Prompt:       "Test prompt",
				AllowedTools: []string{"mcp__cortex__listTickets", "mcp__cortex__readTicket"},
			},
			contains: []string{"--allowedTools", "mcp__cortex__listTickets,mcp__cortex__readTicket"},
		},
		{
			name: "with append system prompt content",
			params: ClaudeCommandParams{
				Prompt:             "Dynamic content",
				AppendSystemPrompt: "## Instructions\nDo the task",
				MCPConfigPath:      "/config.json",
			},
			contains: []string{"--append-system-prompt", "'## Instructions\nDo the task'", "'Dynamic content'"},
		},
		{
			name: "with append system prompt content containing quotes",
			params: ClaudeCommandParams{
				Prompt:             "Dynamic content",
				AppendSystemPrompt: "It's a test prompt",
				MCPConfigPath:      "/config.json",
			},
			contains: []string{"--append-system-prompt", "'It'\\''s a test prompt'"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := BuildClaudeCommand(tc.params)
			for _, expected := range tc.contains {
				if !contains(cmd, expected) {
					t.Errorf("expected command to contain %q, got: %s", expected, cmd)
				}
			}
		})
	}
}

func TestEscapePromptForShell(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"it's", "it'\\''s"},
		{"don't do it", "don'\\''t do it"},
		{"'''", "'\\'''\\'''\\''"}, // Each ' becomes '\''
	}

	for _, tc := range tests {
		result := EscapePromptForShell(tc.input)
		if result != tc.expected {
			t.Errorf("EscapePromptForShell(%q): expected %q, got: %q", tc.input, tc.expected, result)
		}
	}
}

func TestStateInfo_CanSpawn(t *testing.T) {
	tests := []struct {
		state    SessionState
		expected bool
	}{
		{StateNormal, true},
		{StateActive, false},
		{StateOrphaned, true},
		{StateEnded, true},
	}

	for _, tc := range tests {
		info := &StateInfo{State: tc.state}
		if info.CanSpawn() != tc.expected {
			t.Errorf("CanSpawn() for %s: expected %v", tc.state, tc.expected)
		}
	}
}

func TestStateInfo_CanResume(t *testing.T) {
	tests := []struct {
		name     string
		info     StateInfo
		expected bool
	}{
		{"orphaned with session ID", StateInfo{State: StateOrphaned, Session: &ticket.Session{ID: "abc"}}, true},
		{"orphaned without session", StateInfo{State: StateOrphaned}, false},
		{"orphaned with empty session ID", StateInfo{State: StateOrphaned, Session: &ticket.Session{ID: ""}}, false},
		{"active", StateInfo{State: StateActive, Session: &ticket.Session{ID: "abc"}}, false},
		{"normal", StateInfo{State: StateNormal}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.info.CanResume() != tc.expected {
				t.Errorf("expected %v", tc.expected)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
