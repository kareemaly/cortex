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

// RoleConfig holds configuration for a specific role (architect or ticket type).
type RoleConfig struct {
	Agent     AgentType `yaml:"agent"`
	Args      []string  `yaml:"args"`
	Companion string    `yaml:"companion,omitempty"`
}

// TicketsConfig holds configuration for the ticket storage.
type TicketsConfig struct {
	Path string `yaml:"path,omitempty"`
}

// Config holds the project configuration.
type Config struct {
	Name      string        `yaml:"name"`
	Repos     []string      `yaml:"repos,omitempty"`
	Architect RoleConfig    `yaml:"architect"`
	Work      RoleConfig    `yaml:"work"`
	Research  RoleConfig    `yaml:"research"`
	Tickets   TicketsConfig `yaml:"tickets,omitempty"`
}

// TicketsPath returns the resolved tickets directory path for the given project root.
// If Tickets.Path is set, resolves it relative to the project root (or absolute).
// Otherwise defaults to {projectRoot}/tickets.
func (c *Config) TicketsPath(projectRoot string) string {
	if c.Tickets.Path != "" {
		if filepath.IsAbs(c.Tickets.Path) {
			return c.Tickets.Path
		}
		return filepath.Join(projectRoot, c.Tickets.Path)
	}
	return filepath.Join(projectRoot, "tickets")
}

// SessionsPath returns the resolved sessions directory path for the given project root.
// Defaults to {projectRoot}/sessions.
func (c *Config) SessionsPath(projectRoot string) string {
	return filepath.Join(projectRoot, "sessions")
}

// RoleConfigForType returns the RoleConfig for a given ticket type.
// Returns an error if the ticket type is not "work" or "research".
func (c *Config) RoleConfigForType(ticketType string) (RoleConfig, error) {
	switch ticketType {
	case "work":
		return c.Work, nil
	case "research":
		return c.Research, nil
	default:
		return RoleConfig{}, fmt.Errorf("no configuration found for ticket type %q (valid types: work, research)", ticketType)
	}
}

// ValidateRepo checks if a repo is in the project's repos list.
// If Repos is empty, any repo is allowed.
func (c *Config) ValidateRepo(repo string) error {
	if len(c.Repos) == 0 {
		return nil
	}
	for _, r := range c.Repos {
		if r == repo {
			return nil
		}
	}
	return fmt.Errorf("repo %q not in project repos list", repo)
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		Architect: RoleConfig{
			Agent:     AgentClaude,
			Companion: "cortex project",
		},
		Work: RoleConfig{
			Agent:     AgentClaude,
			Companion: "cortex ticket show",
		},
		Research: RoleConfig{
			Agent: AgentClaude,
		},
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

// Load loads configuration from the project root.
// Tries {projectRoot}/cortex.yaml first, falls back to {projectRoot}/.cortex/cortex.yaml.
// Returns default config if neither file exists.
func Load(projectRoot string) (*Config, error) {
	absPath, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, err
	}

	// Try cortex.yaml at project root first
	configPath := filepath.Join(absPath, "cortex.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Fall back to .cortex/cortex.yaml
			configPath = filepath.Join(absPath, ".cortex", "cortex.yaml")
			data, err = os.ReadFile(configPath)
			if err != nil {
				if os.IsNotExist(err) {
					return DefaultConfig(), nil
				}
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	cfg := DefaultConfig()
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
	// Validate architect agent type
	if c.Architect.Agent != "" && c.Architect.Agent != AgentClaude && c.Architect.Agent != AgentOpenCode {
		return &ValidationError{
			Field:   "architect.agent",
			Message: "must be 'claude' or 'opencode'",
		}
	}

	// Validate work agent type
	if c.Work.Agent != "" && c.Work.Agent != AgentClaude && c.Work.Agent != AgentOpenCode {
		return &ValidationError{
			Field:   "work.agent",
			Message: "must be 'claude' or 'opencode'",
		}
	}

	// Validate research agent type
	if c.Research.Agent != "" && c.Research.Agent != AgentClaude && c.Research.Agent != AgentOpenCode {
		return &ValidationError{
			Field:   "research.agent",
			Message: "must be 'claude' or 'opencode'",
		}
	}

	return nil
}
