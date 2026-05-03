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

	"github.com/hiveryn/agentruntime"
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
	sessions        map[string]*session.Session // keyed by SessionID UUID
	createErr       error
	endErr          error
	endCalls        []string // tracks which session IDs had End called
	lastCreateAgent string
}

// addTicketSession pre-populates a ticket session for tests that need an
// existing session without going through Create. Returns the SessionID.
func (m *mockSessionStore) addTicketSession(ticketID, tmuxWindow string) string {
	sess := &session.Session{
		SessionID:  session.NewSessionID(),
		Type:       session.SessionTypeTicket,
		TicketID:   ticketID,
		TmuxWindow: tmuxWindow,
		StartedAt:  time.Now(),
	}
	m.sessions[sess.SessionID] = sess
	return sess.SessionID
}

func newMockSessionStore() *mockSessionStore {
	return &mockSessionStore{
		sessions: make(map[string]*session.Session),
	}
}

func (m *mockSessionStore) Create(ticketID, agent, tmuxWindow string) (*session.Session, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	m.lastCreateAgent = agent
	sess := &session.Session{
		SessionID:  session.NewSessionID(),
		Type:       session.SessionTypeTicket,
		TicketID:   ticketID,
		Agent:      agent,
		TmuxWindow: tmuxWindow,
		StartedAt:  time.Now(),
		Status:     session.AgentStatusStarting,
	}
	m.sessions[sess.SessionID] = sess
	return sess, nil
}

func (m *mockSessionStore) EndByTicketID(ticketID string) error {
	if m.endErr != nil {
		return m.endErr
	}
	for id, sess := range m.sessions {
		if sess.Type == session.SessionTypeTicket && sess.TicketID == ticketID {
			m.endCalls = append(m.endCalls, id)
			delete(m.sessions, id)
			return nil
		}
	}
	return nil
}

func (m *mockSessionStore) EndBySessionID(sessionID string) error {
	if m.endErr != nil {
		return m.endErr
	}
	m.endCalls = append(m.endCalls, sessionID)
	delete(m.sessions, sessionID)
	return nil
}

func (m *mockSessionStore) GetByTicketID(ticketID string) (*session.Session, error) {
	for _, sess := range m.sessions {
		if sess.Type == session.SessionTypeTicket && sess.TicketID == ticketID {
			return sess, nil
		}
	}
	return nil, &storage.NotFoundError{Resource: "session", ID: ticketID}
}

func (m *mockSessionStore) CreateArchitect(agent, tmuxWindow string) (*session.Session, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	m.lastCreateAgent = agent
	// Remove any existing architect
	for id, sess := range m.sessions {
		if sess.Type == session.SessionTypeArchitect {
			delete(m.sessions, id)
		}
	}
	sess := &session.Session{
		SessionID:  session.NewSessionID(),
		Type:       session.SessionTypeArchitect,
		Agent:      agent,
		TmuxWindow: tmuxWindow,
		StartedAt:  time.Now(),
		Status:     session.AgentStatusStarting,
	}
	m.sessions[sess.SessionID] = sess
	return sess, nil
}

func (m *mockSessionStore) GetArchitect() (*session.Session, error) {
	for _, sess := range m.sessions {
		if sess.Type == session.SessionTypeArchitect {
			return sess, nil
		}
	}
	return nil, &storage.NotFoundError{Resource: "session", ID: session.ArchitectSessionKey}
}

func (m *mockSessionStore) EndArchitect() error {
	if m.endErr != nil {
		return m.endErr
	}
	for id, sess := range m.sessions {
		if sess.Type == session.SessionTypeArchitect {
			m.endCalls = append(m.endCalls, id)
			delete(m.sessions, id)
			return nil
		}
	}
	return nil
}

func (m *mockSessionStore) CreateCollab(collabID, prompt, agent, tmuxWindow string) (*session.Session, error) {
	sess := &session.Session{
		SessionID:  session.NewSessionID(),
		Type:       session.SessionTypeCollab,
		CollabID:   collabID,
		Prompt:     prompt,
		Agent:      agent,
		TmuxWindow: tmuxWindow,
	}
	m.sessions[sess.SessionID] = sess
	return sess, nil
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
	path := filepath.Join(projectPath, "prompts", relPath)
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

	createTestPromptFile(t, tmpDir, "work/SYSTEM.md", "## Test Instructions")

	spawner := NewSpawner(Dependencies{
		Store:        store,
		SessionStore: sessStore,
		TmuxManager:  tmuxMgr,
		CortexdPath:  "/usr/bin/cortexd",
		MCPConfigDir: tmpDir,
	})

	// Execute
	result, err := spawner.Spawn(context.Background(), SpawnRequest{
		AgentType:     AgentTypeTicketAgent,
		Agent:         "claude",
		TmuxSession:   "test-session",
		ArchitectPath: tmpDir,
		TicketsDir:    filepath.Join(tmpDir, "tickets"),
		TicketID:      "ticket-1",
		Ticket:        testTicket,
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

	// Verify launcher script was created and uses argv-safe exec pattern
	launcherPath := strings.TrimPrefix(tmuxMgr.lastCommand, "bash ")
	data, err := os.ReadFile(launcherPath)
	if err != nil {
		t.Fatalf("failed to read launcher script: %v", err)
	}
	script := string(data)
	if !containsSubstr(script, "args=(") {
		t.Error("expected launcher to use args=( bash array")
	}
	if !containsSubstr(script, `exec "${args[@]}"`) {
		t.Error("expected exec \"${args[@]}\"")
	}
	if !containsSubstr(script, "export CORTEX_TICKET_ID=") {
		t.Error("expected launcher to export CORTEX_TICKET_ID")
	}
}

func TestSpawn_VariantEnv_InLauncherScript(t *testing.T) {
	tmpDir := t.TempDir()
	store := newMockStore()
	sessStore := newMockSessionStore()
	tmuxMgr := newMockTmuxManager()

	testTicket := createTestTicket("ticket-1", "Test Ticket", "Test body")
	store.tickets["ticket-1"] = testTicket
	createTestPromptFile(t, tmpDir, "work/SYSTEM.md", "## Test Instructions")

	spawner := NewSpawner(Dependencies{
		Store:        store,
		SessionStore: sessStore,
		TmuxManager:  tmuxMgr,
		CortexdPath:  "/usr/bin/cortexd",
		MCPConfigDir: tmpDir,
	})

	result, err := spawner.Spawn(context.Background(), SpawnRequest{
		AgentType:     AgentTypeTicketAgent,
		Agent:         "claude",
		TmuxSession:   "test-session",
		ArchitectPath: tmpDir,
		TicketsDir:    filepath.Join(tmpDir, "tickets"),
		TicketID:      "ticket-1",
		Ticket:        testTicket,
		EnvVars: map[string]string{
			"CODEX_HOME":    "/Users/me/.codex-personal",
			"MY_CUSTOM_VAR": "hello world",
		},
	})
	if err != nil || !result.Success {
		t.Fatalf("spawn failed: err=%v result=%v", err, result)
	}

	launcherPath := strings.TrimPrefix(tmuxMgr.lastCommand, "bash ")
	data, err := os.ReadFile(launcherPath)
	if err != nil {
		t.Fatalf("failed to read launcher script: %v", err)
	}
	script := string(data)

	if !containsSubstr(script, "export CODEX_HOME='/Users/me/.codex-personal'") {
		t.Errorf("expected CODEX_HOME variant env in launcher; script:\n%s", script)
	}
	if !containsSubstr(script, "export MY_CUSTOM_VAR='hello world'") {
		t.Errorf("expected MY_CUSTOM_VAR variant env in launcher; script:\n%s", script)
	}
}

func TestSpawn_CodexVariantCODEXHomeSeedsTempConfigButDoesNotOverride(t *testing.T) {
	tmpDir := t.TempDir()
	store := newMockStore()
	sessStore := newMockSessionStore()
	tmuxMgr := newMockTmuxManager()

	sourceHome := filepath.Join(tmpDir, "codex-personal")
	if err := os.MkdirAll(sourceHome, 0755); err != nil {
		t.Fatal(err)
	}
	sourceAuth := filepath.Join(sourceHome, "auth.json")
	if err := os.WriteFile(sourceAuth, []byte(`{"profile":"personal"}`), 0600); err != nil {
		t.Fatal(err)
	}

	testTicket := createTestTicket("ticket-1", "Test Ticket", "Test body")
	store.tickets["ticket-1"] = testTicket
	createTestPromptFile(t, tmpDir, "work/SYSTEM.md", "## Test Instructions")

	spawner := NewSpawner(Dependencies{
		Store:        store,
		SessionStore: sessStore,
		TmuxManager:  tmuxMgr,
		CortexdPath:  "/usr/bin/cortexd",
		MCPConfigDir: tmpDir,
	})

	result, err := spawner.Spawn(context.Background(), SpawnRequest{
		AgentType:     AgentTypeTicketAgent,
		Agent:         "codex",
		TmuxSession:   "test-session",
		ArchitectPath: tmpDir,
		TicketsDir:    filepath.Join(tmpDir, "tickets"),
		TicketID:      "ticket-1",
		Ticket:        testTicket,
		EnvVars: map[string]string{
			"CODEX_HOME":    sourceHome,
			"MY_CUSTOM_VAR": "hello world",
		},
	})
	if err != nil || !result.Success {
		t.Fatalf("spawn failed: err=%v result=%v", err, result)
	}

	launcherPath := strings.TrimPrefix(tmuxMgr.lastCommand, "bash ")
	data, err := os.ReadFile(launcherPath)
	if err != nil {
		t.Fatalf("failed to read launcher script: %v", err)
	}
	script := string(data)
	codexHome := extractExportedEnvVar(t, script, "CODEX_HOME")

	if codexHome == "" {
		t.Error("expected CODEX_HOME to be set in env; script has no CODEX_HOME export")
	} else if codexHome != sourceHome {
		t.Errorf("expected variant CODEX_HOME=%q to flow through, got %q; script:\n%s", sourceHome, codexHome, script)
	}
	if !containsSubstr(script, "export MY_CUSTOM_VAR='hello world'") {
		t.Errorf("expected non-CODEX_HOME variant env to remain in launcher; script:\n%s", script)
	}

	// With agentruntime, codex uses --config flags instead of config.toml.
	if !containsSubstr(script, "--config") {
		t.Errorf("expected codex --config flags for MCP/instructions; script:\n%s", script)
	}
	if containsSubstr(script, "config.toml") {
		t.Errorf("codex no longer uses config.toml; uses --config flags; script:\n%s", script)
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
	_ = sessStore.addTicketSession("ticket-1", "test-window-prepop")

	spawner := NewSpawner(Dependencies{
		Store:        store,
		SessionStore: sessStore,
		TmuxManager:  tmuxMgr,
		CortexdPath:  "/usr/bin/cortexd",
	})

	// Execute
	result, err := spawner.Spawn(context.Background(), SpawnRequest{
		AgentType:     AgentTypeTicketAgent,
		Agent:         "claude",
		TmuxSession:   "test-session",
		ArchitectPath: tmpDir,
		TicketsDir:    filepath.Join(tmpDir, "tickets"),
		TicketID:      "ticket-1",
		Ticket:        testTicket,
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

	createTestPromptFile(t, tmpDir, "work/SYSTEM.md", "## Test Instructions")

	spawner := NewSpawner(Dependencies{
		Store:        store,
		SessionStore: sessStore,
		TmuxManager:  tmuxMgr,
		CortexdPath:  "/usr/bin/cortexd",
		MCPConfigDir: tmpDir,
	})

	// Execute
	result, err := spawner.Spawn(context.Background(), SpawnRequest{
		AgentType:     AgentTypeTicketAgent,
		Agent:         "claude",
		TmuxSession:   "test-session",
		ArchitectPath: tmpDir,
		TicketsDir:    filepath.Join(tmpDir, "tickets"),
		TicketID:      "ticket-1",
		Ticket:        testTicket,
	})

	// Verify
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure")
	}

	// Session should have been cleaned up (End called). endCalls holds the
	// SessionID UUID in our UUID-keyed mock — just assert at least one call.
	if len(sessStore.endCalls) == 0 {
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
		AgentType:     AgentTypeTicketAgent,
		Agent:         "claude",
		TmuxSession:   "test-session",
		ArchitectPath: tmpDir,
		TicketsDir:    filepath.Join(tmpDir, "tickets"),
		WindowName:    "test-ticket",
		TicketID:      "ticket-1",
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
	existingID := sessStore.addTicketSession("ticket-1", "old-window")

	createTestPromptFile(t, tmpDir, "work/SYSTEM.md", "## Test Instructions")

	spawner := NewSpawner(Dependencies{
		Store:        store,
		SessionStore: sessStore,
		TmuxManager:  tmuxMgr,
		CortexdPath:  "/usr/bin/cortexd",
		MCPConfigDir: tmpDir,
	})

	// Execute
	result, err := spawner.Fresh(context.Background(), SpawnRequest{
		AgentType:     AgentTypeTicketAgent,
		Agent:         "claude",
		TmuxSession:   "test-session",
		ArchitectPath: tmpDir,
		TicketsDir:    filepath.Join(tmpDir, "tickets"),
		TicketID:      "ticket-1",
		Ticket:        testTicket,
	})

	// Verify
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Message)
	}

	// End should have been called for the old session's SessionID.
	if !slices.Contains(sessStore.endCalls, existingID) {
		t.Errorf("expected End to be called for old session %q, endCalls=%v", existingID, sessStore.endCalls)
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

func TestBuildMCPServerConfig_WithTicket(t *testing.T) {
	config := BuildMCPServerConfig(MCPConfigParams{
		CortexdPath:   "/usr/bin/cortexd",
		TicketID:      "ticket-123",
		TicketType:    "work",
		TicketsDir:    "/path/to/tickets",
		ArchitectPath: "/path/to/project",
		TmuxSession:   "dev-session",
	})

	if config.Name != "cortex" {
		t.Errorf("expected name 'cortex', got: %s", config.Name)
	}

	if config.Command != "/usr/bin/cortexd" {
		t.Errorf("expected command '/usr/bin/cortexd', got: %s", config.Command)
	}

	expectedArgs := []string{"mcp", "--ticket-id", "ticket-123", "--ticket-type", "work"}
	if len(config.Args) != len(expectedArgs) {
		t.Fatalf("expected %d args, got: %d (%v)", len(expectedArgs), len(config.Args), config.Args)
	}
	for i, arg := range expectedArgs {
		if config.Args[i] != arg {
			t.Errorf("arg[%d]: expected %s, got: %s", i, arg, config.Args[i])
		}
	}

	if config.Env["CORTEX_TICKETS_DIR"] != "/path/to/tickets" {
		t.Errorf("expected CORTEX_TICKETS_DIR, got: %v", config.Env)
	}

	if config.Env["CORTEX_DAEMON_URL"] != daemonconfig.DefaultDaemonURL {
		t.Errorf("expected CORTEX_DAEMON_URL=%s, got: %v", daemonconfig.DefaultDaemonURL, config.Env["CORTEX_DAEMON_URL"])
	}
}

func TestBuildMCPServerConfig_WithTicketNoType(t *testing.T) {
	config := BuildMCPServerConfig(MCPConfigParams{
		CortexdPath:   "/usr/bin/cortexd",
		TicketID:      "ticket-123",
		TicketsDir:    "/path/to/tickets",
		ArchitectPath: "/path/to/project",
		TmuxSession:   "dev-session",
	})

	expectedArgs := []string{"mcp", "--ticket-id", "ticket-123"}
	if len(config.Args) != len(expectedArgs) {
		t.Fatalf("expected %d args, got: %d (%v)", len(expectedArgs), len(config.Args), config.Args)
	}
	for i, arg := range expectedArgs {
		if config.Args[i] != arg {
			t.Errorf("arg[%d]: expected %s, got: %s", i, arg, config.Args[i])
		}
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
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		t.Errorf("unexpected error removing prompt file: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected prompt file to be removed")
	}
}

func TestWriteLauncherScript(t *testing.T) {
	tmpDir := t.TempDir()

	spec := agentruntime.LaunchSpec{
		Command: "claude",
		Args: []string{
			"prompt text here",
			"--append-system-prompt", "system instructions",
			"--mcp-config", "/tmp/cortex-mcp-test.json",
			"--session-id", "session-abc",
			"--permission-mode", "plan",
		},
		Env: map[string]string{
			"AGENTRUNTIME_SESSION_ID": "session-abc",
		},
		Workdir: "/tmp/work",
		CleanupPaths: []string{
			"/tmp/cortex-mcp-test.json",
		},
	}
	extraEnv := map[string]string{
		"CORTEX_TICKET_ID": "ticket-1",
	}

	path, err := WriteLauncherScript(spec, extraEnv, "test-id", tmpDir)
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
	if !containsSubstr(script, "AGENTRUNTIME_SESSION_ID") {
		t.Error("expected AGENTRUNTIME_SESSION_ID export")
	}

	// Verify argv-safe exec pattern
	if !containsSubstr(script, "args=(") {
		t.Error("expected args=( bash array")
	}
	if !containsSubstr(script, `exec "${args[@]}"`) {
		t.Error("expected exec \"${args[@]}\"")
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

func TestWriteLauncherScript_VariantEnv(t *testing.T) {
	tmpDir := t.TempDir()

	spec := agentruntime.LaunchSpec{
		Command: "claude",
		Args:    []string{"--mcp-config", "/tmp/cortex-mcp-test.json", "--session-id", "abc"},
		Env: map[string]string{
			"CODEX_HOME":    "/Users/me/.codex-personal",
			"MY_CUSTOM_VAR": "hello world",
		},
		CleanupPaths: []string{"/tmp/cortex-mcp-test.json"},
	}
	extraEnv := map[string]string{
		"CORTEX_TICKET_ID": "ticket-1",
	}

	path, err := WriteLauncherScript(spec, extraEnv, "variant-env-test", tmpDir)
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

	// Verify trap with cleanup
	if !containsSubstr(script, "trap 'rm -f") {
		t.Error("expected trap for cleanup")
	}

	// Verify argv-safe exec pattern
	if !containsSubstr(script, "args=(") {
		t.Error("expected args=( bash array")
	}
	if !containsSubstr(script, `exec "${args[@]}"`) {
		t.Error("expected exec \"${args[@]}\"")
	}

	// All three env vars must be exported
	if !containsSubstr(script, "export CORTEX_TICKET_ID='ticket-1'") {
		t.Errorf("expected CORTEX_TICKET_ID export; script:\n%s", script)
	}
	if !containsSubstr(script, "export CODEX_HOME='/Users/me/.codex-personal'") {
		t.Errorf("expected CODEX_HOME export; script:\n%s", script)
	}
	if !containsSubstr(script, "export MY_CUSTOM_VAR='hello world'") {
		t.Errorf("expected MY_CUSTOM_VAR export; script:\n%s", script)
	}
}

func TestWriteLauncherScript_Resume(t *testing.T) {
	tmpDir := t.TempDir()

	spec := agentruntime.LaunchSpec{
		Command: "claude",
		Args: []string{
			"--resume", "session-to-resume",
			"--mcp-config", "/tmp/cortex-mcp-test.json",
			"--permission-mode", "plan",
		},
		CleanupPaths: []string{"/tmp/cortex-mcp-test.json"},
	}

	path, err := WriteLauncherScript(spec, nil, "resume-test", tmpDir)
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

	// Verify trap with cleanup
	if !containsSubstr(script, "trap 'rm -f") {
		t.Error("expected trap for cleanup")
	}

	// Verify argv-safe exec pattern
	if !containsSubstr(script, "args=(") {
		t.Error("expected args=( bash array")
	}
	if !containsSubstr(script, `exec "${args[@]}"`) {
		t.Error("expected exec \"${args[@]}\"")
	}

	// Should have --resume flag with the session ID
	if !containsSubstr(script, "'--resume' 'session-to-resume'") {
		t.Error("expected '--resume' 'session-to-resume' in args array; script:\n" + script)
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

func TestWriteLauncherScript_BareResume(t *testing.T) {
	tmpDir := t.TempDir()

	spec := agentruntime.LaunchSpec{
		Command:      "claude",
		Args:         []string{"--resume", "--mcp-config", "/tmp/cortex-mcp-test.json"},
		CleanupPaths: []string{"/tmp/cortex-mcp-test.json"},
	}

	path, err := WriteLauncherScript(spec, nil, "bare-resume-test", tmpDir)
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

	// Verify trap with cleanup
	if !containsSubstr(script, "trap 'rm -f") {
		t.Error("expected trap for cleanup")
	}

	// Verify argv-safe exec pattern
	if !containsSubstr(script, "args=(") {
		t.Error("expected args=( bash array")
	}
	if !containsSubstr(script, `exec "${args[@]}"`) {
		t.Error("expected exec \"${args[@]}\"")
	}

	// Should have bare --resume flag
	if !containsSubstr(script, "'--resume'") {
		t.Error("expected '--resume' in args array")
	}

	// Should NOT have --resume followed by a space+ID (bare resume)
	if containsSubstr(script, "'--resume' 'session") {
		t.Error("expected bare --resume without a specific ID")
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

func TestWriteLauncherScript_Architect(t *testing.T) {
	tmpDir := t.TempDir()

	spec := agentruntime.LaunchSpec{
		Command: "claude",
		Args: []string{
			"prompt text here",
			"--system-prompt", "system instructions",
			"--mcp-config", "/tmp/cortex-mcp-arch.json",
			"--session-id", "arch-session",
			"--allowedTools", "mcp__cortex__listTickets,mcp__cortex__readTicket",
		},
		CleanupPaths: []string{
			"/tmp/cortex-mcp-arch.json",
			"/tmp/cortex-prompt-arch.txt",
			"/tmp/cortex-sysprompt-arch.txt",
		},
	}

	path, err := WriteLauncherScript(spec, nil, "architect", tmpDir)
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

	// Verify trap with cleanup
	if !containsSubstr(script, "trap 'rm -f") {
		t.Error("expected trap for cleanup")
	}

	// Verify argv-safe exec pattern
	if !containsSubstr(script, "args=(") {
		t.Error("expected args=( bash array")
	}
	if !containsSubstr(script, `exec "${args[@]}"`) {
		t.Error("expected exec \"${args[@]}\"")
	}

	// Should have --allowedTools via shell-quoted args
	if !containsSubstr(script, "'--allowedTools' 'mcp__cortex__listTickets,mcp__cortex__readTicket'") {
		t.Error("expected '--allowedTools' 'tools' via shell-quoted args")
	}

	// Should use --system-prompt (full replace), NOT --append-system-prompt
	if !containsSubstr(script, "'--system-prompt'") {
		t.Error("expected '--system-prompt' flag for architect")
	}
	if containsSubstr(script, "'--append-system-prompt'") {
		t.Error("architect should use --system-prompt, not --append-system-prompt")
	}

	// Should NOT have ticket env vars
	if containsSubstr(script, "CORTEX_TICKET_ID") {
		t.Error("architect should not have CORTEX_TICKET_ID")
	}

	// Should NOT have --permission-mode (architect doesn't use plan mode)
	if containsSubstr(script, "'--permission-mode'") {
		t.Error("architect should not have --permission-mode")
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
		AgentType:     AgentTypeTicketAgent,
		TmuxSession:   "test-session",
		ArchitectPath: "/nonexistent/path/that/does/not/exist",
		TicketID:      "ticket-1",
		Ticket:        createTestTicket("ticket-1", "Test", "Body"),
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
				AgentType:     AgentTypeTicketAgent,
				TmuxSession:   tc.tmuxSession,
				ArchitectPath: tmpDir,
				TicketID:      "ticket-1",
				Ticket:        createTestTicket("ticket-1", "Test", "Body"),
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
	createTestPromptFile(t, tmpDir, "work/SYSTEM.md", "## Test Instructions")

	return tmpDir, store, sessStore, tmuxMgr
}

func TestOrchestrate_Normal_Normal(t *testing.T) {
	tmpDir, store, sessStore, tmuxMgr := orchestrateTestSetup(t)

	result, err := Orchestrate(context.Background(), OrchestrateRequest{
		TicketID:      "ticket-1",
		Mode:          "normal",
		ArchitectPath: tmpDir,
		TmuxSession:   "test-session",
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
	_ = sessStore.addTicketSession("ticket-1", "test-window-prepop")

	result, err := Orchestrate(context.Background(), OrchestrateRequest{
		TicketID:      "ticket-1",
		Mode:          "normal",
		ArchitectPath: tmpDir,
		TmuxSession:   "test-session",
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
	_ = sessStore.addTicketSession("ticket-1", "test-window-prepop")

	_, err := Orchestrate(context.Background(), OrchestrateRequest{
		TicketID:      "ticket-1",
		Mode:          "normal",
		ArchitectPath: tmpDir,
		TmuxSession:   "test-session",
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
		TicketID:      "ticket-1",
		Mode:          "resume",
		ArchitectPath: tmpDir,
		TmuxSession:   "test-session",
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

	_ = sessStore.addTicketSession("ticket-1", "test-window-prepop")

	_, err := Orchestrate(context.Background(), OrchestrateRequest{
		TicketID:      "ticket-1",
		Mode:          "resume",
		ArchitectPath: tmpDir,
		TmuxSession:   "test-session",
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

	_ = sessStore.addTicketSession("ticket-1", "test-window-prepop")

	result, err := Orchestrate(context.Background(), OrchestrateRequest{
		TicketID:      "ticket-1",
		Mode:          "resume",
		ArchitectPath: tmpDir,
		TmuxSession:   "test-session",
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
		TicketID:      "ticket-1",
		Mode:          "fresh",
		ArchitectPath: tmpDir,
		TmuxSession:   "test-session",
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

	_ = sessStore.addTicketSession("ticket-1", "test-window-prepop")

	_, err := Orchestrate(context.Background(), OrchestrateRequest{
		TicketID:      "ticket-1",
		Mode:          "fresh",
		ArchitectPath: tmpDir,
		TmuxSession:   "test-session",
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

	_ = sessStore.addTicketSession("ticket-1", "test-window-prepop")

	result, err := Orchestrate(context.Background(), OrchestrateRequest{
		TicketID:      "ticket-1",
		Mode:          "fresh",
		ArchitectPath: tmpDir,
		TmuxSession:   "test-session",
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
		TicketID:      "ticket-1",
		Mode:          "normal",
		ArchitectPath: tmpDir,
		TmuxSession:   "test-session",
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
		TicketID:      "ticket-1",
		Mode:          "normal",
		ArchitectPath: tmpDir,
		TmuxSession:   "test-session",
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
		TicketID:      "ticket-1",
		Mode:          "",
		ArchitectPath: tmpDir,
		TmuxSession:   "test-session",
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
		TicketID:      "ticket-1",
		Mode:          "invalid",
		ArchitectPath: tmpDir,
		TmuxSession:   "test-session",
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

// extractExportedEnvVar extracts the value of a shell-exported env var from a launcher script.
// The launcher writes lines like: export VAR='value'
// This reverses the shellQuote escaping (backslash-quote sequences).
func extractExportedEnvVar(t *testing.T, script, varName string) string {
	t.Helper()
	prefix := "export " + varName + "='"
	idx := strings.Index(script, prefix)
	if idx == -1 {
		t.Fatalf("could not find %q in script", prefix)
	}
	// Find the value between the outer single quotes.
	// shellQuote produces 'val' where inner ' becomes '\''
	// We scan forward from after the opening quote, handling '\'' sequences.
	start := idx + len(prefix)
	var sb strings.Builder
	i := start
	for i < len(script) {
		if script[i] == '\'' {
			// Check for '\'' escape sequence (end-quote, escaped-quote, start-quote)
			if i+3 < len(script) && script[i:i+4] == "'\\''" {
				sb.WriteByte('\'')
				i += 4
				continue
			}
			// Closing quote
			break
		}
		sb.WriteByte(script[i])
		i++
	}
	return sb.String()
}
