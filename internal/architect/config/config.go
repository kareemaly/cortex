package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
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

// ResearchRoleConfig holds configuration for the research role, extending RoleConfig with paths.
type ResearchRoleConfig struct {
	RoleConfig `yaml:",inline"`
	Paths      []string `yaml:"paths,omitempty"`
}

// TicketsConfig holds configuration for the ticket storage.
type TicketsConfig struct {
	Path string `yaml:"path,omitempty"`
}

// Config holds the architect configuration.
type Config struct {
	Name      string             `yaml:"name"`
	Repos     []string           `yaml:"repos,omitempty"`
	Architect RoleConfig         `yaml:"architect"`
	Work      RoleConfig         `yaml:"work"`
	Research  ResearchRoleConfig `yaml:"research"`
	Collab    RoleConfig         `yaml:"collab,omitempty"`
	Tickets   TicketsConfig      `yaml:"tickets,omitempty"`
}

// TicketsPath returns the resolved tickets directory path for the given architect root.
// If Tickets.Path is set, resolves it relative to the architect root (or absolute).
// Otherwise defaults to {architectRoot}/tickets.
func (c *Config) TicketsPath(architectRoot string) string {
	if c.Tickets.Path != "" {
		if filepath.IsAbs(c.Tickets.Path) {
			return c.Tickets.Path
		}
		return filepath.Join(architectRoot, c.Tickets.Path)
	}
	return filepath.Join(architectRoot, "tickets")
}

// SessionsPath returns the resolved sessions directory path for the given architect root.
// Defaults to {architectRoot}/sessions.
func (c *Config) SessionsPath(architectRoot string) string {
	return filepath.Join(architectRoot, "sessions")
}

// GetTmuxSessionName returns the tmux session name for this architect.
// Uses Config.Name if set, otherwise defaults to "cortex".
func (c *Config) GetTmuxSessionName() string {
	if c.Name != "" {
		return c.Name
	}
	return "cortex"
}

// RoleConfigForType returns the RoleConfig for a given ticket type.
// Returns an error if the ticket type is not "work" or "research".
func (c *Config) RoleConfigForType(ticketType string) (RoleConfig, error) {
	switch ticketType {
	case "work":
		return c.Work, nil
	case "research":
		return c.Research.RoleConfig, nil
	default:
		return RoleConfig{}, fmt.Errorf("no configuration found for ticket type %q (valid types: work, research)", ticketType)
	}
}

// expandHome expands a leading ~/ to the user's home directory.
func expandHome(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// ValidateRepo checks if a repo is in the architect's repos list.
// If Repos is empty, any repo is allowed.
func (c *Config) ValidateRepo(repo string) error {
	if len(c.Repos) == 0 {
		return nil
	}
	expanded := expandHome(repo)
	for _, r := range c.Repos {
		if expandHome(r) == expanded {
			return nil
		}
	}
	return fmt.Errorf("repo %q not in architect repos list", repo)
}

// ValidateResearchPath checks if a path is allowed for research tickets.
// Checks exact match against repos and glob match against research.paths.
// If both repos and research.paths are empty, any path is allowed.
func (c *Config) ValidateResearchPath(path string) error {
	expanded := expandHome(path)

	// If no restrictions configured, allow all
	if len(c.Repos) == 0 && len(c.Research.Paths) == 0 {
		return nil
	}

	// Check exact match against repos
	for _, r := range c.Repos {
		if expandHome(r) == expanded {
			return nil
		}
	}

	// Check glob match against research.paths
	for _, glob := range c.Research.Paths {
		matched, err := doublestar.Match(expandHome(glob), expanded)
		if err == nil && matched {
			return nil
		}
	}

	return fmt.Errorf("path %q not allowed; must match a configured repo or research.paths entry", path)
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		Architect: RoleConfig{
			Agent:     AgentClaude,
			Companion: "cortex architect show",
		},
		Work: RoleConfig{
			Agent: AgentClaude,
		},
		Research: ResearchRoleConfig{
			RoleConfig: RoleConfig{
				Agent: AgentClaude,
			},
		},
		Collab: RoleConfig{
			Agent: AgentClaude,
		},
	}
}

// FindArchitectRoot walks up from startPath to find a cortex.yaml file.
// Returns the path containing the cortex.yaml file.
func FindArchitectRoot(startPath string) (string, error) {
	absPath, err := filepath.Abs(startPath)
	if err != nil {
		return "", err
	}

	current := absPath
	for {
		cortexYaml := filepath.Join(current, "cortex.yaml")
		if _, err := os.Stat(cortexYaml); err == nil {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", &ArchitectNotFoundError{StartPath: startPath}
		}
		current = parent
	}
}

// Load loads configuration from cortex.yaml at the architect root.
// Returns default config if no cortex.yaml exists.
func Load(architectRoot string) (*Config, error) {
	absPath, err := filepath.Abs(architectRoot)
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(absPath, "cortex.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
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

// LoadFromPath finds the architect root from the given path and loads config.
// Returns the config, architect root path, and any error.
func LoadFromPath(path string) (*Config, string, error) {
	architectRoot, err := FindArchitectRoot(path)
	if err != nil {
		return nil, "", err
	}

	cfg, err := Load(architectRoot)
	if err != nil {
		return nil, "", err
	}

	return cfg, architectRoot, nil
}

// ConfigPath returns the path to cortex.yaml for the given architect root.
func ConfigPath(architectRoot string) string {
	absPath, err := filepath.Abs(architectRoot)
	if err != nil {
		absPath = architectRoot
	}
	return filepath.Join(absPath, "cortex.yaml")
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

	// Validate collab agent type
	if c.Collab.Agent != "" && c.Collab.Agent != AgentClaude && c.Collab.Agent != AgentOpenCode {
		return &ValidationError{
			Field:   "collab.agent",
			Message: "must be 'claude' or 'opencode'",
		}
	}

	return nil
}
