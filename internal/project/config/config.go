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
	Agent AgentType `yaml:"agent"`
	Args  []string  `yaml:"args"`
}

// TicketConfig maps ticket type names to their role configurations.
type TicketConfig map[string]RoleConfig

// Config holds the project configuration.
type Config struct {
	Extend    string       `yaml:"extend,omitempty"`
	Name      string       `yaml:"name"`
	Architect RoleConfig   `yaml:"architect"`
	Ticket    TicketConfig `yaml:"ticket"`
	Git       GitConfig    `yaml:"git"`

	// resolvedExtendPath is the absolute path of the resolved extend directory.
	// Set during Load() if Extend is specified.
	resolvedExtendPath string
}

// ResolvedExtendPath returns the resolved absolute path of the extend directory,
// or empty string if no extend is configured.
func (c *Config) ResolvedExtendPath() string {
	return c.resolvedExtendPath
}

// GitConfig holds git-related configuration.
type GitConfig struct {
	Worktrees bool `yaml:"worktrees"`
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		Architect: RoleConfig{Agent: AgentClaude},
		Ticket: TicketConfig{
			"work": RoleConfig{Agent: AgentClaude},
		},
	}
}

// TicketRoleConfig returns the RoleConfig for a given ticket type.
// Returns an error if the ticket type is not configured.
func (c *Config) TicketRoleConfig(ticketType string) (RoleConfig, error) {
	if c.Ticket == nil {
		return RoleConfig{}, fmt.Errorf("no ticket configuration found for type %q", ticketType)
	}
	role, ok := c.Ticket[ticketType]
	if !ok {
		return RoleConfig{}, fmt.Errorf("no ticket configuration found for type %q", ticketType)
	}
	return role, nil
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
// If the config has an extend field, it recursively loads and merges the base config.
func Load(projectRoot string) (*Config, error) {
	return loadWithVisited(projectRoot, make(map[string]bool))
}

// loadWithVisited loads config with circular reference detection.
func loadWithVisited(projectRoot string, visited map[string]bool) (*Config, error) {
	absPath, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, err
	}

	// Check for circular reference
	if visited[absPath] {
		return nil, &CircularExtendError{Path: absPath}
	}
	visited[absPath] = true

	configPath := filepath.Join(projectRoot, ".cortex", "cortex.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}

	// Parse raw config (without merging defaults yet)
	var rawCfg Config
	if err := yaml.Unmarshal(data, &rawCfg); err != nil {
		return nil, &ConfigParseError{Path: configPath, Err: err}
	}

	// If no extend, apply defaults and return
	if rawCfg.Extend == "" {
		cfg := DefaultConfig()
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, &ConfigParseError{Path: configPath, Err: err}
		}
		if err := cfg.Validate(); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	// Resolve and validate extend path
	resolvedExtendPath, err := ValidateExtendPath(rawCfg.Extend, projectRoot)
	if err != nil {
		return nil, err
	}

	// Recursively load base config
	baseCfg, err := loadWithVisited(resolvedExtendPath, visited)
	if err != nil {
		return nil, err
	}

	// Merge: base first, then overlay project config
	merged := MergeConfigs(baseCfg, &rawCfg)
	merged.resolvedExtendPath = resolvedExtendPath

	if err := merged.Validate(); err != nil {
		return nil, err
	}

	return merged, nil
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

	// Validate each ticket type's agent
	for typeName, role := range c.Ticket {
		if role.Agent != "" && role.Agent != AgentClaude && role.Agent != AgentOpenCode {
			return &ValidationError{
				Field:   fmt.Sprintf("ticket.%s.agent", typeName),
				Message: "must be 'claude' or 'opencode'",
			}
		}
	}

	return nil
}
