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
architect:
  agent: opencode
  args:
    - "--verbose"
ticket:
  work:
    agent: opencode
    args:
      - "--budget=50000"
  investigation:
    agent: claude
    args:
      - "--fast"
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
	if cfg.Architect.Agent != AgentOpenCode {
		t.Errorf("expected architect agent 'opencode', got %q", cfg.Architect.Agent)
	}
	if len(cfg.Architect.Args) != 1 || cfg.Architect.Args[0] != "--verbose" {
		t.Errorf("expected architect args [--verbose], got %v", cfg.Architect.Args)
	}
	workRole, err := cfg.TicketRoleConfig("work")
	if err != nil {
		t.Fatalf("unexpected error getting work role: %v", err)
	}
	if workRole.Agent != AgentOpenCode {
		t.Errorf("expected work agent 'opencode', got %q", workRole.Agent)
	}
	if len(workRole.Args) != 1 || workRole.Args[0] != "--budget=50000" {
		t.Errorf("expected work args [--budget=50000], got %v", workRole.Args)
	}
	invRole, err := cfg.TicketRoleConfig("investigation")
	if err != nil {
		t.Fatalf("unexpected error getting investigation role: %v", err)
	}
	if invRole.Agent != AgentClaude {
		t.Errorf("expected investigation agent 'claude', got %q", invRole.Agent)
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
	if cfg.Architect.Agent != AgentClaude {
		t.Errorf("expected default architect agent 'claude', got %q", cfg.Architect.Agent)
	}
	workRole, err := cfg.TicketRoleConfig("work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if workRole.Agent != AgentClaude {
		t.Errorf("expected default work agent 'claude', got %q", workRole.Agent)
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
	if cfg.Architect.Agent != AgentClaude {
		t.Errorf("expected default architect agent 'claude', got %q", cfg.Architect.Agent)
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

func TestValidate_InvalidArchitectAgent(t *testing.T) {
	cfg := &Config{
		Architect: RoleConfig{Agent: "invalid-agent"},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}

	valErr := err.(*ValidationError)
	if valErr.Field != "architect.agent" {
		t.Errorf("expected field 'architect.agent', got %q", valErr.Field)
	}
}

func TestValidate_InvalidTicketAgent(t *testing.T) {
	cfg := &Config{
		Ticket: TicketConfig{
			"work": RoleConfig{Agent: "bad-agent"},
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
	if valErr.Field != "ticket.work.agent" {
		t.Errorf("expected field 'ticket.work.agent', got %q", valErr.Field)
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

func TestNestedConfig_ArchitectAndTicket(t *testing.T) {
	projectRoot := setupTestProject(t)
	writeConfig(t, projectRoot, `
name: test
architect:
  agent: claude
  args:
    - "--budget=150000"
    - "--verbose"
ticket:
  work:
    agent: claude
    args:
      - "--budget=50000"
`)

	cfg, err := Load(projectRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Architect.Args) != 2 {
		t.Fatalf("expected 2 architect args, got %d", len(cfg.Architect.Args))
	}
	if cfg.Architect.Args[0] != "--budget=150000" {
		t.Errorf("expected --budget=150000, got %q", cfg.Architect.Args[0])
	}
	if cfg.Architect.Args[1] != "--verbose" {
		t.Errorf("expected --verbose, got %q", cfg.Architect.Args[1])
	}

	workRole, err := cfg.TicketRoleConfig("work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(workRole.Args) != 1 {
		t.Fatalf("expected 1 work arg, got %d", len(workRole.Args))
	}
	if workRole.Args[0] != "--budget=50000" {
		t.Errorf("expected --budget=50000, got %q", workRole.Args[0])
	}
}

func TestNestedConfig_NoArgs(t *testing.T) {
	projectRoot := setupTestProject(t)
	writeConfig(t, projectRoot, `
name: test
`)

	cfg, err := Load(projectRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Architect.Args) != 0 {
		t.Errorf("expected empty architect args, got %v", cfg.Architect.Args)
	}
}

func TestNestedConfig_MultipleTicketTypes(t *testing.T) {
	projectRoot := setupTestProject(t)
	writeConfig(t, projectRoot, `
name: test
ticket:
  work:
    agent: claude
    args:
      - "--budget=50000"
  investigation:
    agent: opencode
    args:
      - "--fast"
`)

	cfg, err := Load(projectRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	workRole, err := cfg.TicketRoleConfig("work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if workRole.Agent != AgentClaude {
		t.Errorf("expected work agent 'claude', got %q", workRole.Agent)
	}

	invRole, err := cfg.TicketRoleConfig("investigation")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if invRole.Agent != AgentOpenCode {
		t.Errorf("expected investigation agent 'opencode', got %q", invRole.Agent)
	}
}

func TestTicketRoleConfig_MissingType(t *testing.T) {
	cfg := &Config{
		Ticket: TicketConfig{
			"work": RoleConfig{Agent: AgentClaude},
		},
	}

	_, err := cfg.TicketRoleConfig("nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestTicketRoleConfig_NilTicketConfig(t *testing.T) {
	cfg := &Config{}

	_, err := cfg.TicketRoleConfig("work")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
