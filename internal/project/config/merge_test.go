package config

import (
	"testing"
)

func TestMergeConfigs_NilInputs(t *testing.T) {
	t.Run("nil base returns project", func(t *testing.T) {
		project := &Config{Name: "project"}
		result := MergeConfigs(nil, project)
		if result != project {
			t.Error("expected project config returned for nil base")
		}
	})

	t.Run("nil project returns base", func(t *testing.T) {
		base := &Config{Name: "base"}
		result := MergeConfigs(base, nil)
		if result != base {
			t.Error("expected base config returned for nil project")
		}
	})

	t.Run("both nil returns nil", func(t *testing.T) {
		result := MergeConfigs(nil, nil)
		if result != nil {
			t.Error("expected nil for both nil inputs")
		}
	})
}

func TestMergeConfigs_ScalarOverride(t *testing.T) {
	t.Run("project name wins", func(t *testing.T) {
		base := &Config{Name: "base-name"}
		project := &Config{Name: "project-name"}

		result := MergeConfigs(base, project)
		if result.Name != "project-name" {
			t.Errorf("expected 'project-name', got %q", result.Name)
		}
	})

	t.Run("base name used when project empty", func(t *testing.T) {
		base := &Config{Name: "base-name"}
		project := &Config{}

		result := MergeConfigs(base, project)
		if result.Name != "base-name" {
			t.Errorf("expected 'base-name', got %q", result.Name)
		}
	})
}

func TestMergeConfigs_ArchitectMerge(t *testing.T) {
	t.Run("project agent wins", func(t *testing.T) {
		base := &Config{Architect: RoleConfig{Agent: AgentClaude}}
		project := &Config{Architect: RoleConfig{Agent: AgentOpenCode}}

		result := MergeConfigs(base, project)
		if result.Architect.Agent != AgentOpenCode {
			t.Errorf("expected 'opencode', got %q", result.Architect.Agent)
		}
	})

	t.Run("base agent used when project empty", func(t *testing.T) {
		base := &Config{Architect: RoleConfig{Agent: AgentOpenCode}}
		project := &Config{}

		result := MergeConfigs(base, project)
		if result.Architect.Agent != AgentOpenCode {
			t.Errorf("expected 'opencode', got %q", result.Architect.Agent)
		}
	})

	t.Run("project args replace base args entirely", func(t *testing.T) {
		base := &Config{
			Architect: RoleConfig{
				Args: []string{"--base-arg1", "--base-arg2"},
			},
		}
		project := &Config{
			Architect: RoleConfig{
				Args: []string{"--project-arg"},
			},
		}

		result := MergeConfigs(base, project)
		if len(result.Architect.Args) != 1 {
			t.Fatalf("expected 1 arg, got %d", len(result.Architect.Args))
		}
		if result.Architect.Args[0] != "--project-arg" {
			t.Errorf("expected '--project-arg', got %q", result.Architect.Args[0])
		}
	})

	t.Run("base args used when project has none", func(t *testing.T) {
		base := &Config{
			Architect: RoleConfig{
				Args: []string{"--base-arg"},
			},
		}
		project := &Config{}

		result := MergeConfigs(base, project)
		if len(result.Architect.Args) != 1 {
			t.Fatalf("expected 1 arg, got %d", len(result.Architect.Args))
		}
		if result.Architect.Args[0] != "--base-arg" {
			t.Errorf("expected '--base-arg', got %q", result.Architect.Args[0])
		}
	})
}

func TestMergeConfigs_TicketMapMerge(t *testing.T) {
	t.Run("base entries preserved", func(t *testing.T) {
		base := &Config{
			Ticket: TicketConfig{
				"work": RoleConfig{Agent: AgentClaude},
			},
		}
		project := &Config{}

		result := MergeConfigs(base, project)
		if _, ok := result.Ticket["work"]; !ok {
			t.Error("expected 'work' ticket type from base")
		}
	})

	t.Run("project entries added", func(t *testing.T) {
		base := &Config{
			Ticket: TicketConfig{
				"work": RoleConfig{Agent: AgentClaude},
			},
		}
		project := &Config{
			Ticket: TicketConfig{
				"investigation": RoleConfig{Agent: AgentOpenCode},
			},
		}

		result := MergeConfigs(base, project)
		if _, ok := result.Ticket["work"]; !ok {
			t.Error("expected 'work' ticket type from base")
		}
		if _, ok := result.Ticket["investigation"]; !ok {
			t.Error("expected 'investigation' ticket type from project")
		}
	})

	t.Run("project entry overrides base on conflict", func(t *testing.T) {
		base := &Config{
			Ticket: TicketConfig{
				"work": RoleConfig{
					Agent: AgentClaude,
					Args:  []string{"--base-arg"},
				},
			},
		}
		project := &Config{
			Ticket: TicketConfig{
				"work": RoleConfig{
					Agent: AgentOpenCode,
					Args:  []string{"--project-arg"},
				},
			},
		}

		result := MergeConfigs(base, project)
		workRole := result.Ticket["work"]
		if workRole.Agent != AgentOpenCode {
			t.Errorf("expected 'opencode', got %q", workRole.Agent)
		}
		if len(workRole.Args) != 1 || workRole.Args[0] != "--project-arg" {
			t.Errorf("expected ['--project-arg'], got %v", workRole.Args)
		}
	})

	t.Run("partial override - agent only", func(t *testing.T) {
		base := &Config{
			Ticket: TicketConfig{
				"work": RoleConfig{
					Agent: AgentClaude,
					Args:  []string{"--base-arg"},
				},
			},
		}
		project := &Config{
			Ticket: TicketConfig{
				"work": RoleConfig{
					Agent: AgentOpenCode,
					// No args - should inherit from base
				},
			},
		}

		result := MergeConfigs(base, project)
		workRole := result.Ticket["work"]
		if workRole.Agent != AgentOpenCode {
			t.Errorf("expected 'opencode', got %q", workRole.Agent)
		}
		if len(workRole.Args) != 1 || workRole.Args[0] != "--base-arg" {
			t.Errorf("expected ['--base-arg'], got %v", workRole.Args)
		}
	})
}

func TestMergeConfigs_GitConfig(t *testing.T) {
	t.Run("project worktrees wins when true", func(t *testing.T) {
		base := &Config{Git: GitConfig{Worktrees: false}}
		project := &Config{Git: GitConfig{Worktrees: true}}

		result := MergeConfigs(base, project)
		if !result.Git.Worktrees {
			t.Error("expected worktrees true")
		}
	})

	t.Run("base worktrees used when project false", func(t *testing.T) {
		base := &Config{Git: GitConfig{Worktrees: true}}
		project := &Config{Git: GitConfig{Worktrees: false}}

		result := MergeConfigs(base, project)
		if !result.Git.Worktrees {
			t.Error("expected worktrees true from base")
		}
	})
}

func TestMergeConfigs_ExtendField(t *testing.T) {
	t.Run("project extend field preserved", func(t *testing.T) {
		base := &Config{Extend: "some/other/path"}
		project := &Config{Extend: "~/.cortex/defaults/basic"}

		result := MergeConfigs(base, project)
		if result.Extend != "~/.cortex/defaults/basic" {
			t.Errorf("expected '~/.cortex/defaults/basic', got %q", result.Extend)
		}
	})
}
