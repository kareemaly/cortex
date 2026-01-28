package spawn

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
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

func (m *mockTmuxManager) SpawnAgent(session, windowName, agentCommand, companionCommand, workingDir, companionWorkingDir string) (int, error) {
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

func (m *mockTmuxManager) SpawnArchitect(session, windowName, agentCommand, companionCommand, workingDir, companionWorkingDir string) error {
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
	result, err := spawner.Spawn(context.Background(), SpawnRequest{
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

	// Verify command uses launcher script
	if !strings.HasPrefix(tmuxMgr.lastCommand, "bash ") {
		t.Errorf("expected command to start with 'bash ', got: %s", tmuxMgr.lastCommand)
	}
	if !containsSubstr(tmuxMgr.lastCommand, "cortex-launcher") {
		t.Errorf("expected command to contain 'cortex-launcher', got: %s", tmuxMgr.lastCommand)
	}

	// Verify launcher script was created and contains expected content
	launcherPath := strings.TrimPrefix(tmuxMgr.lastCommand, "bash ")
	data, err := os.ReadFile(launcherPath)
	if err != nil {
		t.Fatalf("failed to read launcher script: %v", err)
	}
	script := string(data)
	if !containsSubstr(script, "\"$(cat") {
		t.Error("expected launcher to use $(cat) syntax for prompt")
	}
	if containsSubstr(script, "--permission-mode") {
		t.Error("expected no --permission-mode when AgentArgs not provided")
	}
	if !containsSubstr(script, "export CORTEX_TICKET_ID=") {
		t.Error("expected launcher to export CORTEX_TICKET_ID")
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
	result, err := spawner.Spawn(context.Background(), SpawnRequest{
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
	result, err := spawner.Spawn(context.Background(), SpawnRequest{
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
	result, err := spawner.Resume(context.Background(), ResumeRequest{
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

	// Verify command uses launcher script
	if !strings.HasPrefix(tmuxMgr.lastCommand, "bash ") {
		t.Errorf("expected command to start with 'bash ', got: %s", tmuxMgr.lastCommand)
	}
	if !containsSubstr(tmuxMgr.lastCommand, "cortex-launcher") {
		t.Errorf("expected command to contain 'cortex-launcher', got: %s", tmuxMgr.lastCommand)
	}

	// Verify launcher script contains --resume flag
	launcherPath := strings.TrimPrefix(tmuxMgr.lastCommand, "bash ")
	data, err := os.ReadFile(launcherPath)
	if err != nil {
		t.Fatalf("failed to read launcher script: %v", err)
	}
	script := string(data)
	if !containsSubstr(script, "--resume session-abc") {
		t.Error("expected launcher to contain --resume flag")
	}
}

func TestResume_NoSessionID(t *testing.T) {
	spawner := NewSpawner(Dependencies{})

	_, err := spawner.Resume(context.Background(), ResumeRequest{
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

	_, err := spawner.Resume(context.Background(), ResumeRequest{
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
	result, err := spawner.Fresh(context.Background(), SpawnRequest{
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

	if server.Env["CORTEX_DAEMON_URL"] != "http://localhost:4200" {
		t.Errorf("expected CORTEX_DAEMON_URL=http://localhost:4200, got: %v", server.Env["CORTEX_DAEMON_URL"])
	}
}

func TestWritePromptFile(t *testing.T) {
	tmpDir := t.TempDir()

	path, err := WritePromptFile("Hello world prompt", "test-id", "prompt", tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file exists and has correct content
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read prompt file: %v", err)
	}
	if string(data) != "Hello world prompt" {
		t.Errorf("expected content 'Hello world prompt', got: %s", string(data))
	}

	// Verify filename pattern
	if !containsSubstr(path, "cortex-prompt-test-id.txt") {
		t.Errorf("expected path to contain 'cortex-prompt-test-id.txt', got: %s", path)
	}

	// Test remove
	if err := RemovePromptFile(path); err != nil {
		t.Errorf("unexpected error removing prompt file: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected prompt file to be removed")
	}
}

func TestWriteLauncherScript(t *testing.T) {
	tmpDir := t.TempDir()

	params := LauncherParams{
		PromptFilePath:       "/tmp/cortex-prompt-test.txt",
		SystemPromptFilePath: "/tmp/cortex-sysprompt-test.txt",
		MCPConfigPath:        "/tmp/cortex-mcp-test.json",
		SettingsPath:         "/tmp/cortex-settings-test.json",
		SessionID:            "session-abc",
		AgentArgs:            []string{"--permission-mode", "plan"},
		EnvVars: map[string]string{
			"CORTEX_TICKET_ID": "ticket-1",
			"CORTEX_PROJECT":   "/path/to/project",
		},
		CleanupFiles: []string{
			"/tmp/cortex-mcp-test.json",
			"/tmp/cortex-settings-test.json",
			"/tmp/cortex-prompt-test.txt",
			"/tmp/cortex-sysprompt-test.txt",
		},
	}

	path, err := WriteLauncherScript(params, "test-id", tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read launcher script: %v", err)
	}
	script := string(data)

	// Verify shebang
	if !containsSubstr(script, "#!/usr/bin/env bash") {
		t.Error("expected shebang line")
	}

	// Verify trap with cleanup files (should include launcher itself)
	if !containsSubstr(script, "trap 'rm -f") {
		t.Error("expected trap for cleanup")
	}
	if !containsSubstr(script, "cortex-launcher-test-id.sh") {
		t.Error("expected launcher itself in cleanup files")
	}

	// Verify env vars
	if !containsSubstr(script, "export CORTEX_TICKET_ID=") {
		t.Error("expected CORTEX_TICKET_ID export")
	}
	if !containsSubstr(script, "export CORTEX_PROJECT=") {
		t.Error("expected CORTEX_PROJECT export")
	}

	// Verify $(cat) syntax for prompt and system prompt
	if !containsSubstr(script, "\"$(cat") {
		t.Error("expected $(cat) syntax for prompt file")
	}
	if !containsSubstr(script, "--append-system-prompt") {
		t.Error("expected --append-system-prompt flag")
	}

	// Verify other flags
	if !containsSubstr(script, "--mcp-config") {
		t.Error("expected --mcp-config flag")
	}
	if !containsSubstr(script, "--settings") {
		t.Error("expected --settings flag")
	}
	if !containsSubstr(script, "'--permission-mode' 'plan'") {
		t.Error("expected '--permission-mode' 'plan' via AgentArgs")
	}
	if !containsSubstr(script, "--session-id session-abc") {
		t.Error("expected --session-id flag")
	}

	// Verify file is executable
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("failed to stat launcher script: %v", err)
	}
	if info.Mode().Perm()&0100 == 0 {
		t.Error("expected launcher script to be executable")
	}
}

func TestWriteLauncherScript_Resume(t *testing.T) {
	tmpDir := t.TempDir()

	params := LauncherParams{
		MCPConfigPath: "/tmp/cortex-mcp-test.json",
		SettingsPath:  "/tmp/cortex-settings-test.json",
		ResumeID:      "session-to-resume",
		AgentArgs:     []string{"--permission-mode", "plan"},
		CleanupFiles:  []string{"/tmp/cortex-mcp-test.json", "/tmp/cortex-settings-test.json"},
	}

	path, err := WriteLauncherScript(params, "resume-test", tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read launcher script: %v", err)
	}
	script := string(data)

	// Should have --resume flag
	if !containsSubstr(script, "--resume session-to-resume") {
		t.Error("expected --resume flag")
	}

	// Should NOT have $(cat) since there's no prompt file
	if containsSubstr(script, "$(cat") {
		t.Error("did not expect $(cat) syntax for resume (no prompt)")
	}
}

func TestWriteLauncherScript_Architect(t *testing.T) {
	tmpDir := t.TempDir()

	params := LauncherParams{
		PromptFilePath:       "/tmp/cortex-prompt-arch.txt",
		SystemPromptFilePath: "/tmp/cortex-sysprompt-arch.txt",
		ReplaceSystemPrompt:  true,
		MCPConfigPath:        "/tmp/cortex-mcp-arch.json",
		SettingsPath:         "/tmp/cortex-settings-arch.json",
		AgentArgs:            []string{"--allowedTools", "mcp__cortex__listTickets,mcp__cortex__readTicket"},
		SessionID:            "arch-session",
		CleanupFiles: []string{
			"/tmp/cortex-mcp-arch.json",
			"/tmp/cortex-settings-arch.json",
			"/tmp/cortex-prompt-arch.txt",
			"/tmp/cortex-sysprompt-arch.txt",
		},
	}

	path, err := WriteLauncherScript(params, "architect", tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read launcher script: %v", err)
	}
	script := string(data)

	// Should have --allowedTools via AgentArgs (shell-quoted)
	if !containsSubstr(script, "'--allowedTools' 'mcp__cortex__listTickets,mcp__cortex__readTicket'") {
		t.Error("expected '--allowedTools' 'tools' via AgentArgs")
	}

	// Should use --system-prompt (full replace), NOT --append-system-prompt
	if !containsSubstr(script, "--system-prompt") {
		t.Error("expected --system-prompt flag for architect")
	}
	if containsSubstr(script, "--append-system-prompt") {
		t.Error("architect should use --system-prompt, not --append-system-prompt")
	}

	// Should NOT have ticket env vars
	if containsSubstr(script, "CORTEX_TICKET_ID") {
		t.Error("architect should not have CORTEX_TICKET_ID")
	}

	// Should NOT have --permission-mode (architect doesn't use plan mode)
	if containsSubstr(script, "--permission-mode") {
		t.Error("architect should not have --permission-mode")
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
func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestValidateTmuxName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid simple name",
			input:   "my-session",
			wantErr: false,
		},
		{
			name:    "valid with underscores",
			input:   "my_session_123",
			wantErr: false,
		},
		{
			name:    "valid alphanumeric",
			input:   "Session123",
			wantErr: false,
		},
		{
			name:    "starts with hyphen",
			input:   "-session",
			wantErr: true,
			errMsg:  "cannot start with a hyphen",
		},
		{
			name:    "contains colon",
			input:   "my:session",
			wantErr: true,
			errMsg:  "cannot contain colons or periods",
		},
		{
			name:    "contains period",
			input:   "my.session",
			wantErr: true,
			errMsg:  "cannot contain colons or periods",
		},
		{
			name:    "contains space",
			input:   "my session",
			wantErr: true,
			errMsg:  "must contain only alphanumeric",
		},
		{
			name:    "contains special character",
			input:   "my@session",
			wantErr: true,
			errMsg:  "must contain only alphanumeric",
		},
		{
			name:    "too long",
			input:   string(make([]byte, 129)),
			wantErr: true,
			errMsg:  "exceeds maximum length",
		},
		{
			name:    "exactly 128 chars",
			input:   "a" + string(make([]byte, 127)),
			wantErr: true,
			errMsg:  "must contain only alphanumeric", // null bytes
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTmuxName(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tc.errMsg)
				} else if !containsSubstr(err.Error(), tc.errMsg) {
					t.Errorf("expected error containing %q, got %q", tc.errMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateTmuxName_EdgeCases(t *testing.T) {
	// Valid 128 character name
	validLongName := ""
	for range 128 {
		validLongName += "a"
	}
	if err := validateTmuxName(validLongName); err != nil {
		t.Errorf("128 char name should be valid: %v", err)
	}

	// Single character
	if err := validateTmuxName("a"); err != nil {
		t.Errorf("single char should be valid: %v", err)
	}

	// Numeric only
	if err := validateTmuxName("123"); err != nil {
		t.Errorf("numeric only should be valid: %v", err)
	}
}

func TestSpawn_ProjectPathValidation(t *testing.T) {
	spawner := NewSpawner(Dependencies{})

	// Non-existent path
	_, err := spawner.Spawn(context.Background(), SpawnRequest{
		AgentType:   AgentTypeTicketAgent,
		TmuxSession: "test-session",
		ProjectPath: "/nonexistent/path/that/does/not/exist",
		TicketID:    "ticket-1",
		Ticket:      createTestTicket("ticket-1", "Test", "Body"),
	})

	if err == nil {
		t.Fatal("expected error for non-existent project path")
	}
	if !IsConfigError(err) {
		t.Errorf("expected ConfigError, got: %T", err)
	}
	configErr := err.(*ConfigError)
	if configErr.Field != "ProjectPath" {
		t.Errorf("expected Field 'ProjectPath', got: %s", configErr.Field)
	}
}

func TestSpawn_TmuxSessionValidation(t *testing.T) {
	tmpDir := t.TempDir()
	spawner := NewSpawner(Dependencies{})

	tests := []struct {
		name        string
		tmuxSession string
		wantErr     bool
		errField    string
	}{
		{
			name:        "empty session",
			tmuxSession: "",
			wantErr:     true,
			errField:    "TmuxSession",
		},
		{
			name:        "session with colon",
			tmuxSession: "my:session",
			wantErr:     true,
			errField:    "TmuxSession",
		},
		{
			name:        "session starting with hyphen",
			tmuxSession: "-mysession",
			wantErr:     true,
			errField:    "TmuxSession",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := spawner.Spawn(context.Background(), SpawnRequest{
				AgentType:   AgentTypeTicketAgent,
				TmuxSession: tc.tmuxSession,
				ProjectPath: tmpDir,
				TicketID:    "ticket-1",
				Ticket:      createTestTicket("ticket-1", "Test", "Body"),
			})

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if !IsConfigError(err) {
					t.Errorf("expected ConfigError, got: %T", err)
				}
				configErr := err.(*ConfigError)
				if configErr.Field != tc.errField {
					t.Errorf("expected Field %q, got: %s", tc.errField, configErr.Field)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
