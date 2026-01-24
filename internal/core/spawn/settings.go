package spawn

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ClaudeSettingsConfig represents the Claude settings.json structure.
type ClaudeSettingsConfig struct {
	Hooks HooksConfig `json:"hooks"`
}

// HooksConfig defines the hooks section of Claude settings.
type HooksConfig struct {
	PostToolUse       []HookEntry `json:"PostToolUse,omitempty"`
	Stop              []HookEntry `json:"Stop,omitempty"`
	PermissionRequest []HookEntry `json:"PermissionRequest,omitempty"`
}

// HookEntry is a single hook configuration.
type HookEntry struct {
	Matcher string       `json:"matcher,omitempty"`
	Hooks   []HookAction `json:"hooks"`
}

// HookAction defines a hook action.
type HookAction struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// SettingsConfigParams contains parameters for generating settings config.
type SettingsConfigParams struct {
	CortexdPath string
	TicketID    string
	ProjectPath string
}

// GenerateSettingsConfig creates a Claude settings.json config with hooks.
func GenerateSettingsConfig(params SettingsConfigParams) *ClaudeSettingsConfig {
	return &ClaudeSettingsConfig{
		Hooks: HooksConfig{
			PostToolUse: []HookEntry{
				{
					Matcher: "*",
					Hooks: []HookAction{
						{
							Type:    "command",
							Command: params.CortexdPath + " hook post-tool-use",
						},
					},
				},
			},
			Stop: []HookEntry{
				{
					Hooks: []HookAction{
						{
							Type:    "command",
							Command: params.CortexdPath + " hook stop",
						},
					},
				},
			},
			PermissionRequest: []HookEntry{
				{
					Hooks: []HookAction{
						{
							Type:    "command",
							Command: params.CortexdPath + " hook permission-request",
						},
					},
				},
			},
		},
	}
}

// WriteSettingsConfig writes a settings config to a temp file.
// Returns the path to the created file.
func WriteSettingsConfig(config *ClaudeSettingsConfig, identifier, configDir string) (string, error) {
	if configDir == "" {
		configDir = os.TempDir()
	}

	filename := fmt.Sprintf("cortex-settings-%s.json", identifier)
	path := filepath.Join(configDir, filename)

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal settings config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("write settings config: %w", err)
	}

	return path, nil
}

// RemoveSettingsConfig removes a settings config file.
func RemoveSettingsConfig(path string) error {
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove settings config: %w", err)
	}
	return nil
}
