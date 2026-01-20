package config

import (
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

// Config holds the project configuration.
type Config struct {
	Name      string          `yaml:"name"`
	Agent     AgentType       `yaml:"agent"`
	Git       GitConfig       `yaml:"git"`
	Lifecycle LifecycleConfig `yaml:"lifecycle"`
}

// GitConfig holds git-related configuration.
type GitConfig struct {
	Repos []RepoConfig `yaml:"repos"`
}

// RepoConfig holds configuration for a single repository.
type RepoConfig struct {
	Path string `yaml:"path"`
}

// LifecycleConfig holds lifecycle hook configuration.
type LifecycleConfig struct {
	OnPickup  []HookConfig `yaml:"on_pickup"`
	OnSubmit  []HookConfig `yaml:"on_submit"`
	OnApprove []HookConfig `yaml:"on_approve"`
}

// HookConfig holds configuration for a single hook.
type HookConfig struct {
	Run string `yaml:"run"`
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		Agent: AgentClaude,
		Git: GitConfig{
			Repos: []RepoConfig{{Path: "."}},
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

	// Validate repo paths
	for i, repo := range c.Git.Repos {
		if repo.Path == "" {
			return &ValidationError{
				Field:   "git.repos",
				Message: "repo path cannot be empty at index " + itoa(i),
			}
		}
	}

	// Validate lifecycle hooks
	if err := validateHooks("lifecycle.on_pickup", c.Lifecycle.OnPickup); err != nil {
		return err
	}
	if err := validateHooks("lifecycle.on_submit", c.Lifecycle.OnSubmit); err != nil {
		return err
	}
	if err := validateHooks("lifecycle.on_approve", c.Lifecycle.OnApprove); err != nil {
		return err
	}

	return nil
}

func validateHooks(field string, hooks []HookConfig) error {
	for i, hook := range hooks {
		if hook.Run == "" {
			return &ValidationError{
				Field:   field,
				Message: "hook run command cannot be empty at index " + itoa(i),
			}
		}
	}
	return nil
}

// itoa converts an int to string without importing strconv.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var digits []byte
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	return string(digits)
}
