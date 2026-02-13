package spawn

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateOpenCodeStatusPlugin(t *testing.T) {
	plugin := GenerateOpenCodeStatusPlugin(
		"http://localhost:4200",
		"ticket-abc123",
		"/home/user/project",
	)

	// Verify baked-in values
	if !strings.Contains(plugin, `"http://localhost:4200"`) {
		t.Error("expected plugin to contain daemon URL")
	}
	if !strings.Contains(plugin, `"ticket-abc123"`) {
		t.Error("expected plugin to contain ticket ID")
	}
	if !strings.Contains(plugin, `"/home/user/project"`) {
		t.Error("expected plugin to contain project path")
	}

	// Verify event handlers
	for _, event := range []string{
		"session.status",
		"session.idle",
		"permission.asked",
		"permission.replied",
		"tool.execute.before",
		"tool.execute.after",
	} {
		if !strings.Contains(plugin, `"`+event+`"`) {
			t.Errorf("expected plugin to handle event %q", event)
		}
	}

	// Verify status mappings
	if !strings.Contains(plugin, `"in_progress"`) {
		t.Error("expected plugin to map to in_progress status")
	}
	if !strings.Contains(plugin, `"idle"`) {
		t.Error("expected plugin to map to idle status")
	}
	if !strings.Contains(plugin, `"error"`) {
		t.Error("expected plugin to map to error status")
	}
	if !strings.Contains(plugin, `"waiting_permission"`) {
		t.Error("expected plugin to map to waiting_permission status")
	}

	// Verify fire-and-forget with timeout
	if !strings.Contains(plugin, "AbortSignal.timeout(5000)") {
		t.Error("expected plugin to use AbortSignal.timeout(5000)")
	}

	// Verify X-Cortex-Project header
	if !strings.Contains(plugin, "X-Cortex-Project") {
		t.Error("expected plugin to send X-Cortex-Project header")
	}

	// Verify POST to /agent/status
	if !strings.Contains(plugin, "/agent/status") {
		t.Error("expected plugin to POST to /agent/status")
	}
}

func TestWriteOpenCodePluginDir(t *testing.T) {
	pluginContent := "// test plugin content"

	tmpDir, err := WriteOpenCodePluginDir(pluginContent, "test-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Verify temp dir was created
	info, err := os.Stat(tmpDir)
	if err != nil {
		t.Fatalf("temp dir does not exist: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected temp dir to be a directory")
	}

	// Verify plugin file exists at plugin/cortex-status.ts
	pluginPath := filepath.Join(tmpDir, "plugin", "cortex-status.ts")
	data, err := os.ReadFile(pluginPath)
	if err != nil {
		t.Fatalf("failed to read plugin file: %v", err)
	}
	if string(data) != pluginContent {
		t.Errorf("plugin content mismatch: got %q, want %q", string(data), pluginContent)
	}

	// Verify plugin dir is read-only (0555)
	pluginDir := filepath.Join(tmpDir, "plugin")
	dirInfo, err := os.Stat(pluginDir)
	if err != nil {
		t.Fatalf("failed to stat plugin dir: %v", err)
	}
	perm := dirInfo.Mode().Perm()
	if perm != 0555 {
		t.Errorf("expected plugin dir permissions 0555, got %04o", perm)
	}

	// Cleanup: restore write permissions so os.RemoveAll works
	_ = os.Chmod(pluginDir, 0755)
}

func TestOpenCodeSpawnIncludesPluginDir(t *testing.T) {
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

	result, err := spawner.Spawn(t.Context(), SpawnRequest{
		AgentType:   AgentTypeTicketAgent,
		Agent:       "opencode",
		TmuxSession: "test-session",
		ProjectPath: tmpDir,
		TicketsDir:  filepath.Join(tmpDir, "tickets"),
		TicketID:    "ticket-1",
		Ticket:      testTicket,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Message)
	}

	// Read launcher script
	launcherPath := strings.TrimPrefix(tmuxMgr.lastCommand, "bash ")
	data, err := os.ReadFile(launcherPath)
	if err != nil {
		t.Fatalf("failed to read launcher script: %v", err)
	}
	script := string(data)

	// Verify OPENCODE_CONFIG_DIR is exported
	if !containsSubstr(script, "export OPENCODE_CONFIG_DIR=") {
		t.Error("expected OPENCODE_CONFIG_DIR export in launcher script")
	}

	// Verify cleanup trap contains rm -rf for plugin dir
	if !containsSubstr(script, "rm -rf") {
		t.Error("expected rm -rf in trap for plugin dir cleanup")
	}

	// Extract the OPENCODE_CONFIG_DIR value and verify the plugin file exists
	for _, line := range strings.Split(script, "\n") {
		if strings.Contains(line, "export OPENCODE_CONFIG_DIR=") {
			// Extract path between single quotes
			parts := strings.SplitN(line, "'", 3)
			if len(parts) >= 2 {
				configDir := parts[1]
				pluginPath := filepath.Join(configDir, "plugin", "cortex-status.ts")
				pluginData, err := os.ReadFile(pluginPath)
				if err != nil {
					t.Fatalf("failed to read plugin file at %s: %v", pluginPath, err)
				}
				pluginContent := string(pluginData)
				if !strings.Contains(pluginContent, "ticket-1") {
					t.Error("expected plugin to contain ticket ID")
				}
				if !strings.Contains(pluginContent, tmpDir) {
					t.Error("expected plugin to contain project path")
				}
				// Restore write permissions for cleanup
				_ = os.Chmod(filepath.Join(configDir, "plugin"), 0755)
			}
			break
		}
	}
}

func TestClaudeSpawnDoesNotIncludePluginDir(t *testing.T) {
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

	result, err := spawner.Spawn(t.Context(), SpawnRequest{
		AgentType:   AgentTypeTicketAgent,
		Agent:       "claude",
		TmuxSession: "test-session",
		ProjectPath: tmpDir,
		TicketsDir:  filepath.Join(tmpDir, "tickets"),
		TicketID:    "ticket-1",
		Ticket:      testTicket,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Message)
	}

	// Read launcher script
	launcherPath := strings.TrimPrefix(tmuxMgr.lastCommand, "bash ")
	data, err := os.ReadFile(launcherPath)
	if err != nil {
		t.Fatalf("failed to read launcher script: %v", err)
	}
	script := string(data)

	// Verify OPENCODE_CONFIG_DIR is NOT set for claude agent
	if containsSubstr(script, "OPENCODE_CONFIG_DIR") {
		t.Error("claude agent should not have OPENCODE_CONFIG_DIR")
	}
}
