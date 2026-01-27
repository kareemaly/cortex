package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// AgentType represents the type of AI agent to use.
type AgentType string

const (
	AgentClaude   AgentType = "claude"
	AgentOpenCode AgentType = "opencode"
)

// AgentArgsConfig holds separate CLI arguments for architect and ticket sessions.
type AgentArgsConfig struct {
	Architect []string `yaml:"architect"`
	Ticket    []string `yaml:"ticket"`
}

// UnmarshalYAML implements custom YAML unmarshaling for AgentArgsConfig.
// It supports two formats:
//   - New format (mapping): agent_args: {architect: [...], ticket: [...]}
//   - Old format (sequence): agent_args: ["--flag"] — copies to both Architect and Ticket
func (a *AgentArgsConfig) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.MappingNode:
		// New structured format: decode as normal struct
		type plain AgentArgsConfig
		return value.Decode((*plain)(a))
	case yaml.SequenceNode:
		// Old flat format: decode as []string and copy to both fields
		var args []string
		if err := value.Decode(&args); err != nil {
			return err
		}
		a.Architect = args
		a.Ticket = args
		return nil
	default:
		return fmt.Errorf("agent_args must be a mapping or sequence, got %v", value.Kind)
	}
}

// Config holds the project configuration.
type Config struct {
	Name      string          `yaml:"name"`
	Agent     AgentType       `yaml:"agent"`
	AgentArgs AgentArgsConfig `yaml:"agent_args"`
	Git       GitConfig       `yaml:"git"`
}

// GitConfig holds git-related configuration.
type GitConfig struct {
	Worktrees bool `yaml:"worktrees"`
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		Agent: AgentClaude,
	}
}

// FindProjectRoot walks up from startPath to find a .cortex/ directory.
// Returns the path containing the .cortex/ directory.
func FindProjectRoot(startPath string) (string, error) {
	absPath, err := filepath.Abs(startPath)
	if err != nil {
		return "", err
	}

	current := absPath
	for {
		cortexDir := filepath.Join(current, ".cortex")
		info, err := os.Stat(cortexDir)
		if err == nil && info.IsDir() {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached filesystem root
			return "", &ProjectNotFoundError{StartPath: startPath}
		}
		current = parent
	}
}

// Load loads configuration from projectRoot/.cortex/cortex.yaml.
// Returns default config if file doesn't exist.
func Load(projectRoot string) (*Config, error) {
	cfg := DefaultConfig()

	configPath := filepath.Join(projectRoot, ".cortex", "cortex.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, &ConfigParseError{Path: configPath, Err: err}
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// LoadFromPath finds the project root from the given path and loads config.
// Returns the config, project root path, and any error.
func LoadFromPath(path string) (*Config, string, error) {
	projectRoot, err := FindProjectRoot(path)
	if err != nil {
		return nil, "", err
	}

	cfg, err := Load(projectRoot)
	if err != nil {
		return nil, "", err
	}

	return cfg, projectRoot, nil
}

// Validate checks that the config is valid.
func (c *Config) Validate() error {
	// Validate agent type
	if c.Agent != "" && c.Agent != AgentClaude && c.Agent != AgentOpenCode {
		return &ValidationError{
			Field:   "agent",
			Message: "must be 'claude' or 'opencode'",
		}
	}

	return nil
}
