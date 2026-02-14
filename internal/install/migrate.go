package install

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/kareemaly/cortex/internal/daemon/config"
	"gopkg.in/yaml.v3"
)

// MigrationResult holds the result of migrating a single project config.
type MigrationResult struct {
	ProjectPath   string
	ProjectName   string
	DetectedAgent string
	Migrated      bool
	Skipped       bool
	SkipReason    string
	Error         error
}

// legacyConfig is a minimal struct for parsing old configs that may be incomplete.
type legacyConfig struct {
	Name      string `yaml:"name"`
	Extend    string `yaml:"extend,omitempty"`
	Architect struct {
		Agent string `yaml:"agent"`
	} `yaml:"architect,omitempty"`
	Git struct {
		Worktrees bool `yaml:"worktrees"`
	} `yaml:"git,omitempty"`
	Docs struct {
		Path string `yaml:"path,omitempty"`
	} `yaml:"docs,omitempty"`
	Tickets struct {
		Path string `yaml:"path,omitempty"`
	} `yaml:"tickets,omitempty"`
}

// DetectAgentFromExtend returns the agent type based on the extend path.
// Returns "claude" for paths containing "defaults/claude-code",
// "opencode" for paths containing "defaults/opencode", or "" otherwise.
func DetectAgentFromExtend(extendPath string) string {
	if strings.Contains(extendPath, "defaults/claude-code") {
		return "claude"
	}
	if strings.Contains(extendPath, "defaults/opencode") {
		return "opencode"
	}
	return ""
}

// MigrateProjectConfig reads a project's .cortex/cortex.yaml and migrates it
// to the new format with extend pointing to ~/.cortex/defaults/main.
func MigrateProjectConfig(projectPath string) *MigrationResult {
	result := &MigrationResult{ProjectPath: projectPath}

	configPath := filepath.Join(projectPath, ".cortex", "cortex.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			result.Skipped = true
			result.SkipReason = "no .cortex/cortex.yaml found"
			return result
		}
		result.Error = err
		return result
	}

	var old legacyConfig
	if err := yaml.Unmarshal(data, &old); err != nil {
		result.Error = err
		return result
	}

	result.ProjectName = old.Name
	if result.ProjectName == "" {
		result.ProjectName = DetectProjectName(projectPath)
	}

	// Skip if already migrated
	if old.Extend == "~/.cortex/defaults/main" {
		result.Skipped = true
		result.SkipReason = "already using defaults/main"
		return result
	}

	// Detect agent from extend path, fall back to architect.agent
	agent := DetectAgentFromExtend(old.Extend)
	if agent == "" && old.Architect.Agent != "" {
		agent = old.Architect.Agent
	}
	if agent == "" {
		result.Skipped = true
		result.SkipReason = "could not determine agent type"
		return result
	}
	result.DetectedAgent = agent

	// Generate new config
	newConfig := generateProjectConfig(result.ProjectName, agent)

	// Preserve custom fields via string operations
	if old.Git.Worktrees {
		newConfig = strings.Replace(newConfig, "worktrees: false", "worktrees: true", 1)
	}

	// Preserve custom docs/tickets paths
	var extras []string
	if old.Docs.Path != "" {
		extras = append(extras, "docs:\n  path: "+old.Docs.Path)
	}
	if old.Tickets.Path != "" {
		extras = append(extras, "tickets:\n  path: "+old.Tickets.Path)
	}
	if len(extras) > 0 {
		newConfig += strings.Join(extras, "\n") + "\n"
	}

	// Write the migrated config
	if err := os.WriteFile(configPath, []byte(newConfig), 0644); err != nil {
		result.Error = err
		return result
	}

	result.Migrated = true
	return result
}

// MigrateAllProjects loads the global config and migrates all registered projects.
func MigrateAllProjects() ([]MigrationResult, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	var results []MigrationResult
	for _, project := range cfg.Projects {
		r := MigrateProjectConfig(project.Path)
		results = append(results, *r)
	}
	return results, nil
}
