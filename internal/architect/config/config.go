package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	daemonconfig "github.com/kareemaly/cortex/internal/daemon/config"
	"github.com/kareemaly/cortex/internal/storage"
	"gopkg.in/yaml.v3"
)

// AgentType represents the type of AI agent to use.
type AgentType string

const (
	AgentClaude   AgentType = "claude"
	AgentOpenCode AgentType = "opencode"
)

// AgentVariant is a named agent configuration used in the top-level agents map.
type AgentVariant struct {
	Agent AgentType `yaml:"agent"`
	Args  []string  `yaml:"args,omitempty"`
}

// Config holds the architect configuration.
type Config struct {
	Name      string                  `yaml:"name"`
	Queue     bool                    `yaml:"queue,omitempty"`
	Repos     []string                `yaml:"repos,omitempty"`
	Companion string                  `yaml:"companion,omitempty"`
	Agents    map[string]AgentVariant `yaml:"agents,omitempty"`
}

// TicketsPath returns the tickets directory path for the given architect root.
func (c *Config) TicketsPath(architectRoot string) string {
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
		return storage.SanitizeTmuxName(c.Name)
	}
	return "cortex"
}

// ResolveVariant looks up a named variant from the agents map.
// Returns an error if the agents map is empty or the name is not found.
func (c *Config) ResolveVariant(name string) (AgentVariant, error) {
	if len(c.Agents) == 0 {
		return AgentVariant{}, fmt.Errorf("no agents configured — run 'cortex init' to populate defaults in ~/.cortex/settings.yaml")
	}
	v, ok := c.Agents[name]
	if !ok {
		return AgentVariant{}, fmt.Errorf("unknown agent variant %q (available: %v)", name, c.VariantNames())
	}
	return v, nil
}

// MergeAgents merges global agent variants into this config's agents map.
// Global entries are used as a base; project-level entries win on conflict.
func (c *Config) MergeAgents(global map[string]daemonconfig.AgentVariant) {
	if len(global) == 0 {
		return
	}
	if c.Agents == nil {
		c.Agents = make(map[string]AgentVariant)
	}
	for k, v := range global {
		if _, exists := c.Agents[k]; !exists {
			c.Agents[k] = AgentVariant{
				Agent: AgentType(v.Agent),
				Args:  v.Args,
			}
		}
	}
}

// VariantNames returns the sorted list of agent variant names.
// Returns an empty slice if the agents map is nil or empty.
func (c *Config) VariantNames() []string {
	names := make([]string, 0, len(c.Agents))
	for name := range c.Agents {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ValidateRepo checks if a repo is in the architect's repos list.
// If Repos is empty, any repo is allowed.
func (c *Config) ValidateRepo(repo string) error {
	if len(c.Repos) == 0 {
		return nil
	}
	expanded := storage.ExpandHome(repo)
	for _, r := range c.Repos {
		if storage.ExpandHome(r) == expanded {
			return nil
		}
	}
	return fmt.Errorf("repo %q not in architect repos list", repo)
}

// DefaultConfig returns a Config with default values.
// The agents map is intentionally empty; ResolveVariant("default") synthesizes
// a built-in default of {agent: claude} when no agents are configured.
func DefaultConfig() *Config {
	return &Config{}
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
	for name, variant := range c.Agents {
		if variant.Agent != "" && variant.Agent != AgentClaude && variant.Agent != AgentOpenCode {
			return &ValidationError{
				Field:   fmt.Sprintf("agents.%s.agent", name),
				Message: "must be 'claude' or 'opencode'",
			}
		}
	}
	return nil
}
