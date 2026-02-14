package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrateProjectConfig_LegacyClaude(t *testing.T) {
	dir := t.TempDir()
	cortexDir := filepath.Join(dir, ".cortex")
	if err := os.MkdirAll(cortexDir, 0755); err != nil {
		t.Fatal(err)
	}

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
	if err := os.WriteFile(filepath.Join(cortexDir, "cortex.yaml"), []byte(oldConfig), 0644); err != nil {
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

	// Read back and verify
	data, err := os.ReadFile(filepath.Join(cortexDir, "cortex.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "extend: ~/.cortex/defaults/main") {
		t.Error("migrated config should use defaults/main")
	}
	if !strings.Contains(content, "name: myproject") {
		t.Error("migrated config should preserve project name")
	}
}

func TestMigrateProjectConfig_LegacyOpencode(t *testing.T) {
	dir := t.TempDir()
	cortexDir := filepath.Join(dir, ".cortex")
	if err := os.MkdirAll(cortexDir, 0755); err != nil {
		t.Fatal(err)
	}

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
	if err := os.WriteFile(filepath.Join(cortexDir, "cortex.yaml"), []byte(oldConfig), 0644); err != nil {
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

	// Read back and verify worktrees preserved
	data, err := os.ReadFile(filepath.Join(cortexDir, "cortex.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "worktrees: true") {
		t.Error("migrated config should preserve worktrees: true")
	}
}

func TestMigrateProjectConfig_AlreadyMigrated(t *testing.T) {
	dir := t.TempDir()
	cortexDir := filepath.Join(dir, ".cortex")
	if err := os.MkdirAll(cortexDir, 0755); err != nil {
		t.Fatal(err)
	}

	config := `name: myproject
extend: ~/.cortex/defaults/main
architect:
  agent: claude
`
	if err := os.WriteFile(filepath.Join(cortexDir, "cortex.yaml"), []byte(config), 0644); err != nil {
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
	if result.SkipReason != "already using defaults/main" {
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
	cortexDir := filepath.Join(dir, ".cortex")
	if err := os.MkdirAll(cortexDir, 0755); err != nil {
		t.Fatal(err)
	}

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
	if err := os.WriteFile(filepath.Join(cortexDir, "cortex.yaml"), []byte(oldConfig), 0644); err != nil {
		t.Fatal(err)
	}

	result := MigrateProjectConfig(dir)
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if !result.Migrated {
		t.Fatal("expected migration to succeed")
	}

	// Read back and verify custom paths preserved
	data, err := os.ReadFile(filepath.Join(cortexDir, "cortex.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "path: custom/docs") {
		t.Error("migrated config should preserve custom docs path")
	}
	if !strings.Contains(content, "path: custom/tickets") {
		t.Error("migrated config should preserve custom tickets path")
	}
}

func TestMigrateProjectConfig_FallbackToArchitectAgent(t *testing.T) {
	dir := t.TempDir()
	cortexDir := filepath.Join(dir, ".cortex")
	if err := os.MkdirAll(cortexDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Config with no recognizable extend path but has architect.agent
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
	if err := os.WriteFile(filepath.Join(cortexDir, "cortex.yaml"), []byte(oldConfig), 0644); err != nil {
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
