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
	ArchitectPath string
	ArchitectName string
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
	Work struct {
		Agent string `yaml:"agent"`
	} `yaml:"work,omitempty"`
	Git struct {
		Worktrees bool `yaml:"worktrees"`
	} `yaml:"git,omitempty"`
	Docs struct {
		Path string `yaml:"path,omitempty"`
	} `yaml:"docs,omitempty"`
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

// MigrateProjectConfig reads a project's cortex.yaml and migrates it
// from the legacy format (with extend, ticket, git fields) to the new format
// with top-level agents map.
func MigrateProjectConfig(projectPath string) *MigrationResult {
	result := &MigrationResult{ArchitectPath: projectPath}

	configPath := filepath.Join(projectPath, "cortex.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			result.Skipped = true
			result.SkipReason = "no cortex.yaml found"
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

	result.ArchitectName = old.Name
	if result.ArchitectName == "" {
		result.ArchitectName = DetectArchitectName(projectPath)
	}

	// Skip if already migrated (new format has work section and no extend)
	if old.Extend == "" && old.Work.Agent != "" {
		result.Skipped = true
		result.SkipReason = "already using new config format"
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

	// Generate new config with agents variant format
	newConfig := generateMigratedConfig(result.ArchitectName, agent)

	// Write the migrated config
	if err := os.WriteFile(configPath, []byte(newConfig), 0644); err != nil {
		result.Error = err
		return result
	}

	result.Migrated = true
	return result
}

// generateMigratedConfig produces a cortex.yaml in the new agents variant format.
func generateMigratedConfig(name, agent string) string {
	var sb strings.Builder

	sb.WriteString("name: ")
	sb.WriteString(name)
	sb.WriteString("\nrepos: []\n")

	sb.WriteString("\nagents:\n")

	sb.WriteString("  ")
	sb.WriteString(agent)
	sb.WriteString(":\n")
	sb.WriteString("    agent: ")
	sb.WriteString(agent)
	sb.WriteString("\n    args: []\n")

	sb.WriteString("  ")
	sb.WriteString(agent)
	sb.WriteString("-plan:\n")
	sb.WriteString("    agent: ")
	sb.WriteString(agent)
	sb.WriteString("\n")

	if agent == "claude" {
		sb.WriteString("    args: [\"--permission-mode\", \"plan\"]\n")
	} else {
		sb.WriteString("    args: [\"--agent\", \"plan\"]\n")
	}

	return sb.String()
}

// MigrateAllProjects loads the global config and migrates all registered projects.
func MigrateAllProjects() ([]MigrationResult, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	var results []MigrationResult
	for _, project := range cfg.Architects {
		r := MigrateProjectConfig(project.Path)
		results = append(results, *r)
	}
	return results, nil
}
