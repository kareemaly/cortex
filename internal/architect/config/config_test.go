package config

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTestProject creates a temp directory with cortex.yaml at root.
func setupTestProject(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// writeConfig writes a cortex.yaml to the test project.
func writeConfig(t *testing.T, projectRoot, content string) {
	t.Helper()
	configPath := filepath.Join(projectRoot, "cortex.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
}

func TestFindArchitectRoot(t *testing.T) {
	projectRoot := setupTestProject(t)
	writeConfig(t, projectRoot, `name: test`)

	t.Run("finds from root", func(t *testing.T) {
		root, err := FindArchitectRoot(projectRoot)
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

		root, err := FindArchitectRoot(subdir)
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

		root, err := FindArchitectRoot(nested)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if root != projectRoot {
			t.Errorf("expected %s, got %s", projectRoot, root)
		}
	})
}

func TestFindArchitectRoot_NotFound(t *testing.T) {
	// Use a temp directory without cortex.yaml
	dir := t.TempDir()

	_, err := FindArchitectRoot(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsArchitectNotFound(err) {
		t.Errorf("expected ArchitectNotFoundError, got %T", err)
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
work:
  agent: opencode
  args:
    - "--budget=50000"
research:
  agent: claude
  args:
    - "--fast"
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
	workRole, err := cfg.RoleConfigForType("work")
	if err != nil {
		t.Fatalf("unexpected error getting work role: %v", err)
	}
	if workRole.Agent != AgentOpenCode {
		t.Errorf("expected work agent 'opencode', got %q", workRole.Agent)
	}
	if len(workRole.Args) != 1 || workRole.Args[0] != "--budget=50000" {
		t.Errorf("expected work args [--budget=50000], got %v", workRole.Args)
	}
	researchRole, err := cfg.RoleConfigForType("research")
	if err != nil {
		t.Fatalf("unexpected error getting research role: %v", err)
	}
	if researchRole.Agent != AgentClaude {
		t.Errorf("expected research agent 'claude', got %q", researchRole.Agent)
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
	workRole, err := cfg.RoleConfigForType("work")
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
	_, ok := err.(*ConfigParseError)
	if !ok {
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
	valErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if valErr.Field != "architect.agent" {
		t.Errorf("expected field 'architect.agent', got %q", valErr.Field)
	}
}

func TestValidate_InvalidWorkAgent(t *testing.T) {
	cfg := &Config{
		Work: RoleConfig{Agent: "bad-agent"},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	valErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if valErr.Field != "work.agent" {
		t.Errorf("expected field 'work.agent', got %q", valErr.Field)
	}
}

func TestValidate_InvalidResearchAgent(t *testing.T) {
	cfg := &Config{
		Research: ResearchRoleConfig{RoleConfig: RoleConfig{Agent: "bad-agent"}},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	valErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if valErr.Field != "research.agent" {
		t.Errorf("expected field 'research.agent', got %q", valErr.Field)
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
	if !IsArchitectNotFound(err) {
		t.Errorf("expected ArchitectNotFoundError, got %T", err)
	}
}

func TestNestedConfig_ArchitectAndWork(t *testing.T) {
	projectRoot := setupTestProject(t)
	writeConfig(t, projectRoot, `
name: test
architect:
  agent: claude
  args:
    - "--budget=150000"
    - "--verbose"
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

	workRole, err := cfg.RoleConfigForType("work")
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

func TestNestedConfig_WorkAndResearch(t *testing.T) {
	projectRoot := setupTestProject(t)
	writeConfig(t, projectRoot, `
name: test
work:
  agent: claude
  args:
    - "--budget=50000"
research:
  agent: opencode
  args:
    - "--fast"
`)

	cfg, err := Load(projectRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	workRole, err := cfg.RoleConfigForType("work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if workRole.Agent != AgentClaude {
		t.Errorf("expected work agent 'claude', got %q", workRole.Agent)
	}

	researchRole, err := cfg.RoleConfigForType("research")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if researchRole.Agent != AgentOpenCode {
		t.Errorf("expected research agent 'opencode', got %q", researchRole.Agent)
	}
}

func TestRoleConfigForType_InvalidType(t *testing.T) {
	cfg := &Config{
		Work:     RoleConfig{Agent: AgentClaude},
		Research: ResearchRoleConfig{RoleConfig: RoleConfig{Agent: AgentClaude}},
	}

	_, err := cfg.RoleConfigForType("nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRoleConfigForType_DefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	workRole, err := cfg.RoleConfigForType("work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if workRole.Agent != AgentClaude {
		t.Errorf("expected default work agent 'claude', got %q", workRole.Agent)
	}

	researchRole, err := cfg.RoleConfigForType("research")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if researchRole.Agent != AgentClaude {
		t.Errorf("expected default research agent 'claude', got %q", researchRole.Agent)
	}
}

func TestValidateRepo(t *testing.T) {
	t.Run("empty repos allows any", func(t *testing.T) {
		cfg := &Config{}
		if err := cfg.ValidateRepo("any-repo"); err != nil {
			t.Fatalf("expected no error for empty repos list, got: %v", err)
		}
	})

	t.Run("repo in list is allowed", func(t *testing.T) {
		cfg := &Config{
			Repos: []string{"repo-a", "repo-b", "repo-c"},
		}
		if err := cfg.ValidateRepo("repo-b"); err != nil {
			t.Fatalf("expected no error for valid repo, got: %v", err)
		}
	})

	t.Run("repo not in list is rejected", func(t *testing.T) {
		cfg := &Config{
			Repos: []string{"repo-a", "repo-b"},
		}
		err := cfg.ValidateRepo("repo-c")
		if err == nil {
			t.Fatal("expected error for repo not in list, got nil")
		}
	})
}

func TestLoad_WithRepos(t *testing.T) {
	projectRoot := setupTestProject(t)
	writeConfig(t, projectRoot, `
name: multi-repo
repos:
  - frontend
  - backend
  - shared
`)

	cfg, err := Load(projectRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Name != "multi-repo" {
		t.Errorf("expected name 'multi-repo', got %q", cfg.Name)
	}
	if len(cfg.Repos) != 3 {
		t.Fatalf("expected 3 repos, got %d", len(cfg.Repos))
	}
	if cfg.Repos[0] != "frontend" {
		t.Errorf("expected first repo 'frontend', got %q", cfg.Repos[0])
	}
	if err := cfg.ValidateRepo("backend"); err != nil {
		t.Errorf("expected 'backend' to be valid, got: %v", err)
	}
	if err := cfg.ValidateRepo("unknown"); err == nil {
		t.Error("expected error for unknown repo, got nil")
	}
}

func TestGetTmuxSessionName(t *testing.T) {
	tests := []struct {
		name     string
		cfgName  string
		expected string
	}{
		{"simple name", "myproject", "myproject"},
		{"name with spaces", "Footprint Management", "Footprint-Management"},
		{"name with special chars", "my@project#name", "my-project-name"},
		{"name with hyphen", "my-project", "my-project"},
		{"name with underscore", "my_project", "my_project"},
		{"empty name defaults to cortex", "", "cortex"},
		{"name starting with hyphen", "-project", "_project"},
		{"name with multiple spaces", "My  Project  Name", "My--Project--Name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Name: tt.cfgName}
			got := cfg.GetTmuxSessionName()
			if got != tt.expected {
				t.Errorf("GetTmuxSessionName() = %q, want %q", got, tt.expected)
			}
		})
	}
}
