package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultPort is the default port for the cortexd HTTP server.
	DefaultPort = 4200

	// DefaultDaemonURL is the default URL for the cortexd HTTP API.
	DefaultDaemonURL = "http://localhost:4200"
)

// DaemonURL returns the daemon URL for the given port.
func DaemonURL(port int) string {
	return fmt.Sprintf("http://localhost:%d", port)
}

// ProjectEntry represents a registered project in the global config.
type ProjectEntry struct {
	Path  string `yaml:"path"`
	Title string `yaml:"title,omitempty"`
}

// Config holds the daemon configuration.
type Config struct {
	Port               int            `yaml:"port"`
	BindAddress        string         `yaml:"bind_address"`
	LogLevel           string         `yaml:"log_level"`
	StatusHistoryLimit int            `yaml:"status_history_limit"`
	GitDiffTool        string         `yaml:"git_diff_tool"`
	Projects           []ProjectEntry `yaml:"projects,omitempty"`
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		Port:               DefaultPort,
		BindAddress:        "127.0.0.1",
		LogLevel:           "info",
		StatusHistoryLimit: 10,
		GitDiffTool:        "diff",
	}
}

// configPath returns the path to ~/.cortex/settings.yaml.
func configPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".cortex", "settings.yaml"), nil
}

// Load loads configuration from ~/.cortex/settings.yaml.
// Returns default config if file doesn't exist.
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	return LoadFromFile(path)
}

// LoadFromFile loads configuration from the specified file path.
// Returns default config if file doesn't exist.
func LoadFromFile(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save writes the configuration to ~/.cortex/settings.yaml.
func (c *Config) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}
	return c.SaveToFile(path)
}

// SaveToFile writes the configuration to the specified file path.
func (c *Config) SaveToFile(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

// RegisterProject adds a project to the registry if not already present.
// Returns true if the project was newly added.
func (c *Config) RegisterProject(absPath, title string) bool {
	for _, p := range c.Projects {
		if p.Path == absPath {
			return false
		}
	}
	c.Projects = append(c.Projects, ProjectEntry{
		Path:  absPath,
		Title: title,
	})
	return true
}

// UnregisterProject removes a project from the registry.
// Returns true if the project was found and removed.
func (c *Config) UnregisterProject(absPath string) bool {
	for i, p := range c.Projects {
		if p.Path == absPath {
			c.Projects = append(c.Projects[:i], c.Projects[i+1:]...)
			return true
		}
	}
	return false
}
