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
  repos:
    - path: .
    - path: ../shared
lifecycle:
  on_pickup:
    - run: make setup
  on_submit:
    - run: make test
    - run: make lint
  on_approve:
    - run: make deploy
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
	if len(cfg.Git.Repos) != 2 {
		t.Errorf("expected 2 repos, got %d", len(cfg.Git.Repos))
	}
	if cfg.Git.Repos[0].Path != "." {
		t.Errorf("expected first repo path '.', got %q", cfg.Git.Repos[0].Path)
	}
	if cfg.Git.Repos[1].Path != "../shared" {
		t.Errorf("expected second repo path '../shared', got %q", cfg.Git.Repos[1].Path)
	}
	if len(cfg.Lifecycle.OnPickup) != 1 {
		t.Errorf("expected 1 on_pickup hook, got %d", len(cfg.Lifecycle.OnPickup))
	}
	if cfg.Lifecycle.OnPickup[0].Run != "make setup" {
		t.Errorf("expected on_pickup run 'make setup', got %q", cfg.Lifecycle.OnPickup[0].Run)
	}
	if len(cfg.Lifecycle.OnSubmit) != 2 {
		t.Errorf("expected 2 on_submit hooks, got %d", len(cfg.Lifecycle.OnSubmit))
	}
	if len(cfg.Lifecycle.OnApprove) != 1 {
		t.Errorf("expected 1 on_approve hook, got %d", len(cfg.Lifecycle.OnApprove))
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
	if len(cfg.Git.Repos) != 1 || cfg.Git.Repos[0].Path != "." {
		t.Errorf("expected default repos [{'.'}], got %v", cfg.Git.Repos)
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
	if len(cfg.Git.Repos) != 1 || cfg.Git.Repos[0].Path != "." {
		t.Errorf("expected default repos [{'.'}], got %v", cfg.Git.Repos)
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

func TestValidate_EmptyRepoPath(t *testing.T) {
	cfg := &Config{
		Git: GitConfig{
			Repos: []RepoConfig{
				{Path: "."},
				{Path: ""},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}

	valErr := err.(*ValidationError)
	if valErr.Field != "git.repos" {
		t.Errorf("expected field 'git.repos', got %q", valErr.Field)
	}
}

func TestValidate_EmptyHookRun(t *testing.T) {
	tests := []struct {
		name  string
		cfg   *Config
		field string
	}{
		{
			name: "on_pickup",
			cfg: &Config{
				Lifecycle: LifecycleConfig{
					OnPickup: []HookConfig{{Run: ""}},
				},
			},
			field: "lifecycle.on_pickup",
		},
		{
			name: "on_submit",
			cfg: &Config{
				Lifecycle: LifecycleConfig{
					OnSubmit: []HookConfig{{Run: ""}},
				},
			},
			field: "lifecycle.on_submit",
		},
		{
			name: "on_approve",
			cfg: &Config{
				Lifecycle: LifecycleConfig{
					OnApprove: []HookConfig{{Run: ""}},
				},
			},
			field: "lifecycle.on_approve",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T", err)
			}

			valErr := err.(*ValidationError)
			if valErr.Field != tt.field {
				t.Errorf("expected field %q, got %q", tt.field, valErr.Field)
			}
		})
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
