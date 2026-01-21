package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds the daemon configuration.
type Config struct {
	Port               int    `yaml:"port"`
	LogLevel           string `yaml:"log_level"`
	StatusHistoryLimit int    `yaml:"status_history_limit"`
	GitDiffTool        string `yaml:"git_diff_tool"`
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		Port:               4200,
		LogLevel:           "info",
		StatusHistoryLimit: 10,
		GitDiffTool:        "diff",
	}
}

// Load loads configuration from ~/.cortex/settings.yaml.
// Returns default config if file doesn't exist.
func Load() (*Config, error) {
	cfg := DefaultConfig()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".cortex", "settings.yaml")
	data, err := os.ReadFile(configPath)
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
