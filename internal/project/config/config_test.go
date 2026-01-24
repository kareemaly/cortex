package config

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTestProject creates a temp directory with a .cortex/ structure.
func setupTestProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cortexDir := filepath.Join(dir, ".cortex")
	if err := os.Mkdir(cortexDir, 0755); err != nil {
		t.Fatalf("failed to create .cortex dir: %v", err)
	}
	return dir
}

// writeConfig writes a cortex.yaml to the test project.
func writeConfig(t *testing.T, projectRoot, content string) {
	t.Helper()
	configPath := filepath.Join(projectRoot, ".cortex", "cortex.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
}

func TestFindProjectRoot(t *testing.T) {
	projectRoot := setupTestProject(t)

	t.Run("finds from root", func(t *testing.T) {
		root, err := FindProjectRoot(projectRoot)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if root != projectRoot {
			t.Errorf("expected %s, got %s", projectRoot, root)
		}
	})

	t.Run("finds from subdirectory", func(t *testing.T) {
		subdir := filepath.Join(projectRoot, "src")
		if err := os.Mkdir(subdir, 0755); err != nil {
			t.Fatalf("failed to create subdir: %v", err)
		}

		root, err := FindProjectRoot(subdir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if root != projectRoot {
			t.Errorf("expected %s, got %s", projectRoot, root)
		}
	})

	t.Run("finds from nested directory", func(t *testing.T) {
		nested := filepath.Join(projectRoot, "src", "pkg", "internal")
		if err := os.MkdirAll(nested, 0755); err != nil {
			t.Fatalf("failed to create nested dir: %v", err)
		}

		root, err := FindProjectRoot(nested)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if root != projectRoot {
			t.Errorf("expected %s, got %s", projectRoot, root)
		}
	})
}

func TestFindProjectRoot_NotFound(t *testing.T) {
	// Use a temp directory without .cortex/
	dir := t.TempDir()

	_, err := FindProjectRoot(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsProjectNotFound(err) {
		t.Errorf("expected ProjectNotFoundError, got %T", err)
	}
}

func TestLoad_FullConfig(t *testing.T) {
	projectRoot := setupTestProject(t)
	writeConfig(t, projectRoot, `
name: my-project
agent: opencode
git:
  worktrees: true
`)

	cfg, err := Load(projectRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Name != "my-project" {
		t.Errorf("expected name 'my-project', got %q", cfg.Name)
	}
	if cfg.Agent != AgentOpenCode {
		t.Errorf("expected agent 'opencode', got %q", cfg.Agent)
	}
	if !cfg.Git.Worktrees {
		t.Errorf("expected worktrees true, got %v", cfg.Git.Worktrees)
	}
}

func TestLoad_MinimalConfig(t *testing.T) {
	projectRoot := setupTestProject(t)
	writeConfig(t, projectRoot, `
name: minimal
`)

	cfg, err := Load(projectRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Name != "minimal" {
		t.Errorf("expected name 'minimal', got %q", cfg.Name)
	}
	// Should have defaults
	if cfg.Agent != AgentClaude {
		t.Errorf("expected default agent 'claude', got %q", cfg.Agent)
	}
}

func TestLoad_NoConfigFile(t *testing.T) {
	projectRoot := setupTestProject(t)
	// Don't write any config file

	cfg, err := Load(projectRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return defaults
	if cfg.Agent != AgentClaude {
		t.Errorf("expected default agent 'claude', got %q", cfg.Agent)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	projectRoot := setupTestProject(t)
	writeConfig(t, projectRoot, `
name: [invalid yaml
  this is: not valid
`)

	_, err := Load(projectRoot)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsConfigParseError(err) {
		t.Errorf("expected ConfigParseError, got %T: %v", err, err)
	}
}

func TestValidate_InvalidAgent(t *testing.T) {
	cfg := &Config{
		Agent: "invalid-agent",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}

	valErr := err.(*ValidationError)
	if valErr.Field != "agent" {
		t.Errorf("expected field 'agent', got %q", valErr.Field)
	}
}

func TestLoadFromPath(t *testing.T) {
	projectRoot := setupTestProject(t)
	writeConfig(t, projectRoot, `
name: test-project
`)

	subdir := filepath.Join(projectRoot, "cmd", "app")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	cfg, root, err := LoadFromPath(subdir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if root != projectRoot {
		t.Errorf("expected root %s, got %s", projectRoot, root)
	}
	if cfg.Name != "test-project" {
		t.Errorf("expected name 'test-project', got %q", cfg.Name)
	}
}

func TestLoadFromPath_NotFound(t *testing.T) {
	dir := t.TempDir()

	_, _, err := LoadFromPath(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsProjectNotFound(err) {
		t.Errorf("expected ProjectNotFoundError, got %T", err)
	}
}
