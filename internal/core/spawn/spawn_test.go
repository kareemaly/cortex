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

	daemonconfig "github.com/kareemaly/cortex/internal/daemon/config"
	"github.com/kareemaly/cortex/internal/session"
	"github.com/kareemaly/cortex/internal/storage"
	"github.com/kareemaly/cortex/internal/ticket"
)

// mockStore implements StoreInterface for testing (ticket store only).
type mockStore struct {
	tickets map[string]*ticket.Ticket
	getErr  error
}

func newMockStore() *mockStore {
	return &mockStore{
		tickets: make(map[string]*ticket.Ticket),
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

// mockSessionStore implements SessionStoreInterface for testing.
type mockSessionStore struct {
	sessions        map[string]*session.Session // keyed by short ID
	createErr       error
	endErr          error
	endCalls        []string // tracks which short IDs had End called
	lastCreateAgent string
}

func newMockSessionStore() *mockSessionStore {
	return &mockSessionStore{
		sessions: make(map[string]*session.Session),
	}
}

func (m *mockSessionStore) Create(ticketID, agent, tmuxWindow string, worktreePath, featureBranch *string) (string, *session.Session, error) {
	if m.createErr != nil {
		return "", nil, m.createErr
	}
	m.lastCreateAgent = agent
	shortID := storage.ShortID(ticketID)
	sess := &session.Session{
		Type:          session.SessionTypeTicket,
		TicketID:      ticketID,
		Agent:         agent,
		TmuxWindow:    tmuxWindow,
		WorktreePath:  worktreePath,
		FeatureBranch: featureBranch,
		StartedAt:     time.Now(),
		Status:        session.AgentStatusStarting,
	}
	m.sessions[shortID] = sess
	return shortID, sess, nil
}

func (m *mockSessionStore) End(ticketShortID string) error {
	if m.endErr != nil {
		return m.endErr
	}
	m.endCalls = append(m.endCalls, ticketShortID)
	delete(m.sessions, ticketShortID)
	return nil
}

func (m *mockSessionStore) GetByTicketID(ticketID string) (*session.Session, error) {
	shortID := storage.ShortID(ticketID)
	sess, ok := m.sessions[shortID]
	if !ok {
		return nil, &storage.NotFoundError{Resource: "session", ID: shortID}
	}
	return sess, nil
}

func (m *mockSessionStore) CreateArchitect(agent, tmuxWindow string) (*session.Session, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	m.lastCreateAgent = agent
	sess := &session.Session{
		Type:       session.SessionTypeArchitect,
		Agent:      agent,
		TmuxWindow: tmuxWindow,
		StartedAt:  time.Now(),
		Status:     session.AgentStatusStarting,
	}
	m.sessions[session.ArchitectSessionKey] = sess
	return sess, nil
}

func (m *mockSessionStore) GetArchitect() (*session.Session, error) {
	sess, ok := m.sessions[session.ArchitectSessionKey]
	if !ok {
		return nil, &storage.NotFoundError{Resource: "session", ID: session.ArchitectSessionKey}
	}
	return sess, nil
}

func (m *mockSessionStore) EndArchitect() error {
	if m.endErr != nil {
		return m.endErr
	}
	m.endCalls = append(m.endCalls, session.ArchitectSessionKey)
	delete(m.sessions, session.ArchitectSessionKey)
	return nil
}

func (m *mockSessionStore) CreateMeta(agent, tmuxWindow string) (*session.Session, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	m.lastCreateAgent = agent
	sess := &session.Session{
		Type:       session.SessionTypeMeta,
		Agent:      agent,
		TmuxWindow: tmuxWindow,
		StartedAt:  time.Now(),
		Status:     session.AgentStatusStarting,
	}
	m.sessions[session.MetaSessionKey] = sess
	return sess, nil
}

func (m *mockSessionStore) GetMeta() (*session.Session, error) {
	sess, ok := m.sessions[session.MetaSessionKey]
	if !ok {
		return nil, &storage.NotFoundError{Resource: "session", ID: session.MetaSessionKey}
	}
	return sess, nil
}

func (m *mockSessionStore) EndMeta() error {
	if m.endErr != nil {
		return m.endErr
	}
	m.endCalls = append(m.endCalls, session.MetaSessionKey)
	delete(m.sessions, session.MetaSessionKey)
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
		TicketMeta: ticket.TicketMeta{
			ID:    id,
			Title: title,
		},
		Body: body,
	}
}

func createTestPromptFile(t *testing.T, projectPath, relPath, content string) {
	t.Helper()
	path := filepath.Join(projectPath, ".cortex", "prompts", relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// Tests

func TestSpawn_TicketAgent_Success(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	store := newMockStore()
	sessStore := newMockSessionStore()
	tmuxMgr := newMockTmuxManager()

	testTicket := createTestTicket("ticket-1", "Test Ticket", "Test body")
	store.tickets["ticket-1"] = testTicket

	createTestPromptFile(t, tmpDir, "ticket/work/SYSTEM.md", "## Test Instructions")

	spawner := NewSpawner(Dependencies{
		Store:        store,
		SessionStore: sessStore,
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
	sessStore := newMockSessionStore()
	tmuxMgr := newMockTmuxManager()
	tmuxMgr.windowExists = true // Window exists

	testTicket := createTestTicket("ticket-1", "Test Ticket", "Test body")
	store.tickets["ticket-1"] = testTicket

	// Create an existing session
	sessStore.sessions[storage.ShortID("ticket-1")] = &session.Session{
		TicketID:   "ticket-1",
		TmuxWindow: "test-ticket",
		StartedAt:  time.Now(),
	}

	spawner := NewSpawner(Dependencies{
		Store:        store,
		SessionStore: sessStore,
		TmuxManager:  tmuxMgr,
		CortexdPath:  "/usr/bin/cortexd",
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
	sessStore := newMockSessionStore()
	tmuxMgr := newMockTmuxManager()
	tmuxMgr.spawnErr = errors.New("tmux failed")

	testTicket := createTestTicket("ticket-1", "Test Ticket", "Test body")
	store.tickets["ticket-1"] = testTicket

	createTestPromptFile(t, tmpDir, "ticket/work/SYSTEM.md", "## Test Instructions")

	spawner := NewSpawner(Dependencies{
		Store:        store,
		SessionStore: sessStore,
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

	// Session should have been cleaned up (End called)
	shortID := storage.ShortID("ticket-1")
	if !slices.Contains(sessStore.endCalls, shortID) {
		t.Error("expected End to be called for session cleanup")
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

	// Execute - no SessionID means bare --resume (resume most recent)
	result, err := spawner.Resume(context.Background(), ResumeRequest{
		AgentType:   AgentTypeTicketAgent,
		TmuxSession: "test-session",
		ProjectPath: tmpDir,
		TicketsDir:  filepath.Join(tmpDir, "tickets"),
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

	// Verify launcher script contains bare --resume flag (not --resume <id>)
	launcherPath := strings.TrimPrefix(tmuxMgr.lastCommand, "bash ")
	data, err := os.ReadFile(launcherPath)
	if err != nil {
		t.Fatalf("failed to read launcher script: %v", err)
	}
	script := string(data)
	if !containsSubstr(script, "--resume") {
		t.Error("expected launcher to contain --resume flag")
	}
	// Should NOT have --resume followed by a specific ID
	if containsSubstr(script, "--resume ") {
		t.Error("expected bare --resume (no specific ID)")
	}
}

func TestResume_NoTicketID(t *testing.T) {
	spawner := NewSpawner(Dependencies{})

	_, err := spawner.Resume(context.Background(), ResumeRequest{
		AgentType:   AgentTypeTicketAgent,
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
	sessStore := newMockSessionStore()
	tmuxMgr := newMockTmuxManager()

	testTicket := createTestTicket("ticket-1", "Test Ticket", "Test body")
	store.tickets["ticket-1"] = testTicket

	// Create an existing session
	shortID := storage.ShortID("ticket-1")
	sessStore.sessions[shortID] = &session.Session{
		TicketID:   "ticket-1",
		TmuxWindow: "old-window",
		StartedAt:  time.Now(),
	}

	createTestPromptFile(t, tmpDir, "ticket/work/SYSTEM.md", "## Test Instructions")

	spawner := NewSpawner(Dependencies{
		Store:        store,
		SessionStore: sessStore,
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

	// End should have been called for the old session
	if !slices.Contains(sessStore.endCalls, shortID) {
		t.Error("expected End to be called for old session")
	}
}

func TestDetectTicketState_Normal(t *testing.T) {
	// No session = normal state
	info, err := DetectTicketState(nil, "test-session", nil)

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

	sess := &session.Session{
		TicketID:   "ticket-1",
		TmuxWindow: "test-window",
		StartedAt:  time.Now(),
	}

	info, err := DetectTicketState(sess, "test-session", tmuxMgr)

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

	sess := &session.Session{
		TicketID:   "ticket-1",
		TmuxWindow: "test-window",
		StartedAt:  time.Now(),
	}

	info, err := DetectTicketState(sess, "test-session", tmuxMgr)

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

	if server.Env["CORTEX_DAEMON_URL"] != daemonconfig.DefaultDaemonURL {
		t.Errorf("expected CORTEX_DAEMON_URL=%s, got: %v", daemonconfig.DefaultDaemonURL, server.Env["CORTEX_DAEMON_URL"])
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

func TestWriteLauncherScript_BareResume(t *testing.T) {
	tmpDir := t.TempDir()

	params := LauncherParams{
		MCPConfigPath: "/tmp/cortex-mcp-test.json",
		SettingsPath:  "/tmp/cortex-settings-test.json",
		Resume:        true, // bare --resume, no ResumeID
		CleanupFiles:  []string{"/tmp/cortex-mcp-test.json", "/tmp/cortex-settings-test.json"},
	}

	path, err := WriteLauncherScript(params, "bare-resume-test", tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read launcher script: %v", err)
	}
	script := string(data)

	// Should have bare --resume flag
	if !containsSubstr(script, "--resume") {
		t.Error("expected --resume flag")
	}

	// Should NOT have --resume followed by a space+ID
	if containsSubstr(script, "--resume ") {
		t.Error("expected bare --resume without a specific ID")
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
		{"orphaned with session", StateInfo{State: StateOrphaned, Session: &session.Session{TicketID: "abc"}}, true},
		{"orphaned without session", StateInfo{State: StateOrphaned}, false},
		{"active", StateInfo{State: StateActive, Session: &session.Session{TicketID: "abc"}}, false},
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

// --- Orchestrate tests ---

// mockOrchestrateStore implements OrchestrateStore (Get + Move) for testing.
type mockOrchestrateStore struct {
	tickets   map[string]*ticket.Ticket
	statuses  map[string]ticket.Status
	getErr    error
	moveErr   error
	moveCalls []struct {
		ID     string
		Status ticket.Status
	}
}

func newMockOrchestrateStore() *mockOrchestrateStore {
	return &mockOrchestrateStore{
		tickets:  make(map[string]*ticket.Ticket),
		statuses: make(map[string]ticket.Status),
	}
}

func (m *mockOrchestrateStore) Get(id string) (*ticket.Ticket, ticket.Status, error) {
	if m.getErr != nil {
		return nil, "", m.getErr
	}
	t, ok := m.tickets[id]
	if !ok {
		return nil, "", errors.New("ticket not found")
	}
	status := m.statuses[id]
	if status == "" {
		status = ticket.StatusBacklog
	}
	return t, status, nil
}

func (m *mockOrchestrateStore) Move(id string, to ticket.Status) error {
	if m.moveErr != nil {
		return m.moveErr
	}
	m.moveCalls = append(m.moveCalls, struct {
		ID     string
		Status ticket.Status
	}{id, to})
	m.statuses[id] = to
	return nil
}

// orchestrateTestSetup creates common test fixtures for Orchestrate tests.
func orchestrateTestSetup(t *testing.T) (string, *mockOrchestrateStore, *mockSessionStore, *mockTmuxManager) {
	t.Helper()
	tmpDir := t.TempDir()
	store := newMockOrchestrateStore()
	sessStore := newMockSessionStore()
	tmuxMgr := newMockTmuxManager()

	// Add a test ticket
	testTicket := createTestTicket("ticket-1", "Test Ticket", "Test body")
	store.tickets["ticket-1"] = testTicket

	// Create prompt files needed for spawn
	createTestPromptFile(t, tmpDir, "ticket/work/SYSTEM.md", "## Test Instructions")

	return tmpDir, store, sessStore, tmuxMgr
}

func TestOrchestrate_Normal_Normal(t *testing.T) {
	tmpDir, store, sessStore, tmuxMgr := orchestrateTestSetup(t)

	result, err := Orchestrate(context.Background(), OrchestrateRequest{
		TicketID:    "ticket-1",
		Mode:        "normal",
		ProjectPath: tmpDir,
		TmuxSession: "test-session",
	}, OrchestrateDeps{
		Store:        store,
		SessionStore: sessStore,
		TmuxManager:  tmuxMgr,
		CortexdPath:  "/usr/bin/cortexd",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Outcome != OutcomeSpawned {
		t.Errorf("expected OutcomeSpawned, got: %s", result.Outcome)
	}
	if result.SpawnResult == nil || !result.SpawnResult.Success {
		t.Error("expected successful spawn result")
	}
	if result.TmuxSession != "test-session" {
		t.Errorf("expected tmux session 'test-session', got: %s", result.TmuxSession)
	}
}

func TestOrchestrate_Normal_Active(t *testing.T) {
	tmpDir, store, sessStore, tmuxMgr := orchestrateTestSetup(t)
	tmuxMgr.windowExists = true

	// Create an active session
	sessStore.sessions[storage.ShortID("ticket-1")] = &session.Session{
		TicketID:   "ticket-1",
		TmuxWindow: "test-ticket",
		StartedAt:  time.Now(),
	}

	result, err := Orchestrate(context.Background(), OrchestrateRequest{
		TicketID:    "ticket-1",
		Mode:        "normal",
		ProjectPath: tmpDir,
		TmuxSession: "test-session",
	}, OrchestrateDeps{
		Store:        store,
		SessionStore: sessStore,
		TmuxManager:  tmuxMgr,
		CortexdPath:  "/usr/bin/cortexd",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Outcome != OutcomeAlreadyActive {
		t.Errorf("expected OutcomeAlreadyActive, got: %s", result.Outcome)
	}
	if result.SpawnResult != nil {
		t.Error("expected nil SpawnResult for already active")
	}
}

func TestOrchestrate_Normal_Orphaned(t *testing.T) {
	tmpDir, store, sessStore, tmuxMgr := orchestrateTestSetup(t)
	tmuxMgr.windowExists = false

	// Create an orphaned session (session exists, tmux window gone)
	sessStore.sessions[storage.ShortID("ticket-1")] = &session.Session{
		TicketID:   "ticket-1",
		TmuxWindow: "test-ticket",
		StartedAt:  time.Now(),
	}

	_, err := Orchestrate(context.Background(), OrchestrateRequest{
		TicketID:    "ticket-1",
		Mode:        "normal",
		ProjectPath: tmpDir,
		TmuxSession: "test-session",
	}, OrchestrateDeps{
		Store:        store,
		SessionStore: sessStore,
		TmuxManager:  tmuxMgr,
		CortexdPath:  "/usr/bin/cortexd",
	})

	if err == nil {
		t.Fatal("expected StateError for orphaned + normal mode")
	}
	if !IsStateError(err) {
		t.Errorf("expected StateError, got: %T", err)
	}
}

func TestOrchestrate_Resume_Normal(t *testing.T) {
	tmpDir, store, sessStore, tmuxMgr := orchestrateTestSetup(t)

	_, err := Orchestrate(context.Background(), OrchestrateRequest{
		TicketID:    "ticket-1",
		Mode:        "resume",
		ProjectPath: tmpDir,
		TmuxSession: "test-session",
	}, OrchestrateDeps{
		Store:        store,
		SessionStore: sessStore,
		TmuxManager:  tmuxMgr,
		CortexdPath:  "/usr/bin/cortexd",
	})

	if err == nil {
		t.Fatal("expected StateError for resume + normal state")
	}
	if !IsStateError(err) {
		t.Errorf("expected StateError, got: %T", err)
	}
}

func TestOrchestrate_Resume_Active(t *testing.T) {
	tmpDir, store, sessStore, tmuxMgr := orchestrateTestSetup(t)
	tmuxMgr.windowExists = true

	sessStore.sessions[storage.ShortID("ticket-1")] = &session.Session{
		TicketID:   "ticket-1",
		TmuxWindow: "test-ticket",
		StartedAt:  time.Now(),
	}

	_, err := Orchestrate(context.Background(), OrchestrateRequest{
		TicketID:    "ticket-1",
		Mode:        "resume",
		ProjectPath: tmpDir,
		TmuxSession: "test-session",
	}, OrchestrateDeps{
		Store:        store,
		SessionStore: sessStore,
		TmuxManager:  tmuxMgr,
		CortexdPath:  "/usr/bin/cortexd",
	})

	if err == nil {
		t.Fatal("expected StateError for resume + active state")
	}
	if !IsStateError(err) {
		t.Errorf("expected StateError, got: %T", err)
	}
}

func TestOrchestrate_Resume_Orphaned(t *testing.T) {
	tmpDir, store, sessStore, tmuxMgr := orchestrateTestSetup(t)
	tmuxMgr.windowExists = false

	sessStore.sessions[storage.ShortID("ticket-1")] = &session.Session{
		TicketID:   "ticket-1",
		TmuxWindow: "test-ticket",
		StartedAt:  time.Now(),
	}

	result, err := Orchestrate(context.Background(), OrchestrateRequest{
		TicketID:    "ticket-1",
		Mode:        "resume",
		ProjectPath: tmpDir,
		TmuxSession: "test-session",
	}, OrchestrateDeps{
		Store:        store,
		SessionStore: sessStore,
		TmuxManager:  tmuxMgr,
		CortexdPath:  "/usr/bin/cortexd",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Outcome != OutcomeResumed {
		t.Errorf("expected OutcomeResumed, got: %s", result.Outcome)
	}
	if result.SpawnResult == nil || !result.SpawnResult.Success {
		t.Error("expected successful spawn result")
	}
}

func TestOrchestrate_Fresh_Normal(t *testing.T) {
	tmpDir, store, sessStore, tmuxMgr := orchestrateTestSetup(t)

	_, err := Orchestrate(context.Background(), OrchestrateRequest{
		TicketID:    "ticket-1",
		Mode:        "fresh",
		ProjectPath: tmpDir,
		TmuxSession: "test-session",
	}, OrchestrateDeps{
		Store:        store,
		SessionStore: sessStore,
		TmuxManager:  tmuxMgr,
		CortexdPath:  "/usr/bin/cortexd",
	})

	if err == nil {
		t.Fatal("expected StateError for fresh + normal state")
	}
	if !IsStateError(err) {
		t.Errorf("expected StateError, got: %T", err)
	}
}

func TestOrchestrate_Fresh_Active(t *testing.T) {
	tmpDir, store, sessStore, tmuxMgr := orchestrateTestSetup(t)
	tmuxMgr.windowExists = true

	sessStore.sessions[storage.ShortID("ticket-1")] = &session.Session{
		TicketID:   "ticket-1",
		TmuxWindow: "test-ticket",
		StartedAt:  time.Now(),
	}

	_, err := Orchestrate(context.Background(), OrchestrateRequest{
		TicketID:    "ticket-1",
		Mode:        "fresh",
		ProjectPath: tmpDir,
		TmuxSession: "test-session",
	}, OrchestrateDeps{
		Store:        store,
		SessionStore: sessStore,
		TmuxManager:  tmuxMgr,
		CortexdPath:  "/usr/bin/cortexd",
	})

	if err == nil {
		t.Fatal("expected StateError for fresh + active state")
	}
	if !IsStateError(err) {
		t.Errorf("expected StateError, got: %T", err)
	}
}

func TestOrchestrate_Fresh_Orphaned(t *testing.T) {
	tmpDir, store, sessStore, tmuxMgr := orchestrateTestSetup(t)
	tmuxMgr.windowExists = false

	sessStore.sessions[storage.ShortID("ticket-1")] = &session.Session{
		TicketID:   "ticket-1",
		TmuxWindow: "test-ticket",
		StartedAt:  time.Now(),
	}

	result, err := Orchestrate(context.Background(), OrchestrateRequest{
		TicketID:    "ticket-1",
		Mode:        "fresh",
		ProjectPath: tmpDir,
		TmuxSession: "test-session",
	}, OrchestrateDeps{
		Store:        store,
		SessionStore: sessStore,
		TmuxManager:  tmuxMgr,
		CortexdPath:  "/usr/bin/cortexd",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Outcome != OutcomeSpawned {
		t.Errorf("expected OutcomeSpawned, got: %s", result.Outcome)
	}
	if result.SpawnResult == nil || !result.SpawnResult.Success {
		t.Error("expected successful spawn result")
	}
}

func TestOrchestrate_BacklogMovesToProgress(t *testing.T) {
	tmpDir, store, sessStore, tmuxMgr := orchestrateTestSetup(t)
	// Ticket starts in backlog (default)

	result, err := Orchestrate(context.Background(), OrchestrateRequest{
		TicketID:    "ticket-1",
		Mode:        "normal",
		ProjectPath: tmpDir,
		TmuxSession: "test-session",
	}, OrchestrateDeps{
		Store:        store,
		SessionStore: sessStore,
		TmuxManager:  tmuxMgr,
		CortexdPath:  "/usr/bin/cortexd",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Outcome != OutcomeSpawned {
		t.Errorf("expected OutcomeSpawned, got: %s", result.Outcome)
	}

	// Verify ticket was moved to progress
	if len(store.moveCalls) != 1 {
		t.Fatalf("expected 1 Move call, got: %d", len(store.moveCalls))
	}
	if store.moveCalls[0].ID != "ticket-1" {
		t.Errorf("expected Move for ticket-1, got: %s", store.moveCalls[0].ID)
	}
	if store.moveCalls[0].Status != ticket.StatusProgress {
		t.Errorf("expected move to progress, got: %s", store.moveCalls[0].Status)
	}
}

func TestOrchestrate_ProgressDoesNotMove(t *testing.T) {
	tmpDir, store, sessStore, tmuxMgr := orchestrateTestSetup(t)
	// Ticket already in progress
	store.statuses["ticket-1"] = ticket.StatusProgress

	_, err := Orchestrate(context.Background(), OrchestrateRequest{
		TicketID:    "ticket-1",
		Mode:        "normal",
		ProjectPath: tmpDir,
		TmuxSession: "test-session",
	}, OrchestrateDeps{
		Store:        store,
		SessionStore: sessStore,
		TmuxManager:  tmuxMgr,
		CortexdPath:  "/usr/bin/cortexd",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should NOT have moved
	if len(store.moveCalls) != 0 {
		t.Errorf("expected 0 Move calls for already-in-progress ticket, got: %d", len(store.moveCalls))
	}
}

func TestOrchestrate_DefaultModeIsNormal(t *testing.T) {
	tmpDir, store, sessStore, tmuxMgr := orchestrateTestSetup(t)

	// Empty mode should default to "normal" and spawn successfully
	result, err := Orchestrate(context.Background(), OrchestrateRequest{
		TicketID:    "ticket-1",
		Mode:        "",
		ProjectPath: tmpDir,
		TmuxSession: "test-session",
	}, OrchestrateDeps{
		Store:        store,
		SessionStore: sessStore,
		TmuxManager:  tmuxMgr,
		CortexdPath:  "/usr/bin/cortexd",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Outcome != OutcomeSpawned {
		t.Errorf("expected OutcomeSpawned, got: %s", result.Outcome)
	}
}

func TestOrchestrate_InvalidMode(t *testing.T) {
	tmpDir, store, sessStore, tmuxMgr := orchestrateTestSetup(t)

	_, err := Orchestrate(context.Background(), OrchestrateRequest{
		TicketID:    "ticket-1",
		Mode:        "invalid",
		ProjectPath: tmpDir,
		TmuxSession: "test-session",
	}, OrchestrateDeps{
		Store:        store,
		SessionStore: sessStore,
		TmuxManager:  tmuxMgr,
		CortexdPath:  "/usr/bin/cortexd",
	})

	if err == nil {
		t.Fatal("expected ConfigError for invalid mode")
	}
	if !IsConfigError(err) {
		t.Errorf("expected ConfigError, got: %T", err)
	}
}
