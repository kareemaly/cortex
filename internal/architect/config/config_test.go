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
	dir := t.TempDir()

	_, err := FindArchitectRoot(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsArchitectNotFound(err) {
		t.Errorf("expected ArchitectNotFoundError, got %T", err)
	}
}

func TestLoad_NewAgentsSchema(t *testing.T) {
	projectRoot := setupTestProject(t)
	writeConfig(t, projectRoot, `
name: my-project
companion: lazygit
agents:
  default:
    agent: claude
    args:
      - "--dangerously-skip-permissions"
  sonnet:
    agent: claude
    args:
      - "--model"
      - "claude-sonnet-4-6"
      - "--dangerously-skip-permissions"
`)

	cfg, err := Load(projectRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Name != "my-project" {
		t.Errorf("expected name 'my-project', got %q", cfg.Name)
	}
	if cfg.Companion != "lazygit" {
		t.Errorf("expected companion 'lazygit', got %q", cfg.Companion)
	}
	if len(cfg.Agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(cfg.Agents))
	}

	def, err := cfg.ResolveVariant("default")
	if err != nil {
		t.Fatalf("unexpected error resolving default: %v", err)
	}
	if def.Agent != AgentClaude {
		t.Errorf("expected default agent 'claude', got %q", def.Agent)
	}
	if len(def.Args) != 1 || def.Args[0] != "--dangerously-skip-permissions" {
		t.Errorf("unexpected default args: %v", def.Args)
	}

	sonnet, err := cfg.ResolveVariant("sonnet")
	if err != nil {
		t.Fatalf("unexpected error resolving sonnet: %v", err)
	}
	if sonnet.Agent != AgentClaude {
		t.Errorf("expected sonnet agent 'claude', got %q", sonnet.Agent)
	}
	if len(sonnet.Args) != 3 {
		t.Fatalf("expected 3 sonnet args, got %d", len(sonnet.Args))
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
	// Minimal config with no agents map: ResolveVariant errors
	_, err = cfg.ResolveVariant("default")
	if err == nil {
		t.Fatal("expected error when agents map is empty, got nil")
	}
}

func TestLoad_NoConfigFile(t *testing.T) {
	projectRoot := setupTestProject(t)

	cfg, err := Load(projectRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No config file → no agents map → ResolveVariant errors
	_, err = cfg.ResolveVariant("default")
	if err == nil {
		t.Fatal("expected error when no agents configured, got nil")
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

func TestResolveVariant_HappyPath(t *testing.T) {
	cfg := &Config{
		Agents: map[string]AgentVariant{
			"default": {Agent: AgentClaude, Args: []string{"--dangerously-skip-permissions"}},
			"fast":    {Agent: AgentOpenCode},
		},
	}

	v, err := cfg.ResolveVariant("default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Agent != AgentClaude {
		t.Errorf("expected 'claude', got %q", v.Agent)
	}
	if len(v.Args) != 1 {
		t.Errorf("expected 1 arg, got %d", len(v.Args))
	}

	v2, err := cfg.ResolveVariant("fast")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v2.Agent != AgentOpenCode {
		t.Errorf("expected 'opencode', got %q", v2.Agent)
	}
}

func TestResolveVariant_UnknownName(t *testing.T) {
	cfg := &Config{
		Agents: map[string]AgentVariant{
			"default": {Agent: AgentClaude},
		},
	}

	_, err := cfg.ResolveVariant("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown variant, got nil")
	}
}

func TestResolveVariant_EmptyMapErrors(t *testing.T) {
	cfg := &Config{}

	for _, name := range []string{"default", "sonnet", ""} {
		_, err := cfg.ResolveVariant(name)
		if err == nil {
			t.Errorf("expected error for variant %q on empty agents map, got nil", name)
		}
	}
}

func TestVariantNames_Sorted(t *testing.T) {
	cfg := &Config{
		Agents: map[string]AgentVariant{
			"sonnet":  {Agent: AgentClaude},
			"default": {Agent: AgentClaude},
			"alpha":   {Agent: AgentClaude},
		},
	}

	names := cfg.VariantNames()
	if len(names) != 3 {
		t.Fatalf("expected 3 names, got %d", len(names))
	}
	if names[0] != "alpha" || names[1] != "default" || names[2] != "sonnet" {
		t.Errorf("expected sorted [alpha default sonnet], got %v", names)
	}
}

func TestVariantNames_EmptyMapReturnsEmpty(t *testing.T) {
	cfg := &Config{}

	names := cfg.VariantNames()
	if len(names) != 0 {
		t.Errorf("expected empty slice, got %v", names)
	}
}

func TestVariantNames_Populated(t *testing.T) {
	cfg := &Config{
		Agents: map[string]AgentVariant{
			"sonnet":  {Agent: AgentClaude},
			"default": {Agent: AgentClaude},
		},
	}
	names := cfg.VariantNames()
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
	if names[0] != "default" || names[1] != "sonnet" {
		t.Errorf("expected sorted [default sonnet], got %v", names)
	}
}

func TestValidate_InvalidAgentInMap(t *testing.T) {
	cfg := &Config{
		Agents: map[string]AgentVariant{
			"bad": {Agent: "invalid-agent"},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	valErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
	}
	if valErr.Field != "agents.bad.agent" {
		t.Errorf("expected field 'agents.bad.agent', got %q", valErr.Field)
	}
}

func TestValidate_ValidAgents(t *testing.T) {
	cfg := &Config{
		Agents: map[string]AgentVariant{
			"default": {Agent: AgentClaude},
			"oc":      {Agent: AgentOpenCode},
			"noagent": {},
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
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

func TestLoad_WithResearchPaths(t *testing.T) {
	projectRoot := setupTestProject(t)
	writeConfig(t, projectRoot, `
name: test
research:
  paths:
    - ~/projects/**
    - /opt/code/**
`)

	cfg, err := Load(projectRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Research.Paths) != 2 {
		t.Fatalf("expected 2 research paths, got %d", len(cfg.Research.Paths))
	}
}
