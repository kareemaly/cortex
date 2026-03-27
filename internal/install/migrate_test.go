package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrateProjectConfig_LegacyClaude(t *testing.T) {
	dir := t.TempDir()

	oldConfig := `name: myproject
extend: ~/.cortex/defaults/claude-code
architect:
  agent: claude
  args:
    - "--allow-dangerously-skip-permissions"
ticket:
  work:
    agent: claude
git:
  worktrees: false
`
	if err := os.WriteFile(filepath.Join(dir, "cortex.yaml"), []byte(oldConfig), 0644); err != nil {
		t.Fatal(err)
	}

	result := MigrateProjectConfig(dir)
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if !result.Migrated {
		t.Fatal("expected migration to succeed")
	}
	if result.DetectedAgent != "claude" {
		t.Errorf("expected agent 'claude', got %q", result.DetectedAgent)
	}

	data, err := os.ReadFile(filepath.Join(dir, "cortex.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "name: myproject") {
		t.Error("migrated config should preserve project name")
	}
	if !strings.Contains(content, "claude:\n    agent: claude") {
		t.Error("migrated config should have claude variant")
	}
	if !strings.Contains(content, "claude-plan:\n    agent: claude") {
		t.Error("migrated config should have claude-plan variant")
	}
	if strings.Contains(content, "extend:") {
		t.Error("migrated config should not contain extend field")
	}
}

func TestMigrateProjectConfig_LegacyOpencode(t *testing.T) {
	dir := t.TempDir()

	oldConfig := `name: myproject
extend: ~/.cortex/defaults/opencode
architect:
  agent: opencode
ticket:
  work:
    agent: opencode
git:
  worktrees: true
`
	if err := os.WriteFile(filepath.Join(dir, "cortex.yaml"), []byte(oldConfig), 0644); err != nil {
		t.Fatal(err)
	}

	result := MigrateProjectConfig(dir)
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if !result.Migrated {
		t.Fatal("expected migration to succeed")
	}
	if result.DetectedAgent != "opencode" {
		t.Errorf("expected agent 'opencode', got %q", result.DetectedAgent)
	}

	data, err := os.ReadFile(filepath.Join(dir, "cortex.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "name: myproject") {
		t.Error("migrated config should preserve project name")
	}
	if !strings.Contains(content, "opencode:\n    agent: opencode") {
		t.Error("migrated config should have opencode variant")
	}
	if !strings.Contains(content, "opencode-plan:\n    agent: opencode") {
		t.Error("migrated config should have opencode-plan variant")
	}
	if strings.Contains(content, "extend:") {
		t.Error("migrated config should not contain extend field")
	}
	if strings.Contains(content, "worktrees:") {
		t.Error("migrated config should not contain worktrees field")
	}
}

func TestMigrateProjectConfig_AlreadyMigrated(t *testing.T) {
	dir := t.TempDir()

	config := `name: myproject
architect:
  agent: claude
work:
  agent: claude
`
	if err := os.WriteFile(filepath.Join(dir, "cortex.yaml"), []byte(config), 0644); err != nil {
		t.Fatal(err)
	}

	result := MigrateProjectConfig(dir)
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if result.Migrated {
		t.Fatal("should not migrate already-migrated config")
	}
	if !result.Skipped {
		t.Fatal("expected skip")
	}
	if result.SkipReason != "already using new config format" {
		t.Errorf("unexpected skip reason: %s", result.SkipReason)
	}
}

func TestMigrateProjectConfig_NoConfig(t *testing.T) {
	dir := t.TempDir()

	result := MigrateProjectConfig(dir)
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if !result.Skipped {
		t.Fatal("expected skip for missing config")
	}
}

func TestMigrateProjectConfig_CustomPaths(t *testing.T) {
	dir := t.TempDir()

	oldConfig := `name: myproject
extend: ~/.cortex/defaults/claude-code
architect:
  agent: claude
ticket:
  work:
    agent: claude
git:
  worktrees: false
docs:
  path: custom/docs
tickets:
  path: custom/tickets
`
	if err := os.WriteFile(filepath.Join(dir, "cortex.yaml"), []byte(oldConfig), 0644); err != nil {
		t.Fatal(err)
	}

	result := MigrateProjectConfig(dir)
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if !result.Migrated {
		t.Fatal("expected migration to succeed")
	}

	// tickets.path is no longer preserved — tickets always at {root}/tickets
	data, err := os.ReadFile(filepath.Join(dir, "cortex.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if strings.Contains(content, "tickets:") {
		t.Error("migrated config should not contain tickets section")
	}
}

func TestMigrateProjectConfig_FallbackToArchitectAgent(t *testing.T) {
	dir := t.TempDir()

	oldConfig := `name: myproject
extend: /some/custom/path
architect:
  agent: opencode
ticket:
  work:
    agent: opencode
git:
  worktrees: false
`
	if err := os.WriteFile(filepath.Join(dir, "cortex.yaml"), []byte(oldConfig), 0644); err != nil {
		t.Fatal(err)
	}

	result := MigrateProjectConfig(dir)
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if !result.Migrated {
		t.Fatal("expected migration to succeed via architect fallback")
	}
	if result.DetectedAgent != "opencode" {
		t.Errorf("expected agent 'opencode', got %q", result.DetectedAgent)
	}
}
