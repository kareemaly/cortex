package install

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kareemaly/agentstatus"
	_ "github.com/kareemaly/agentstatus/adapters/claude"
	_ "github.com/kareemaly/agentstatus/adapters/codex"
	_ "github.com/kareemaly/agentstatus/adapters/opencode"
	"github.com/kareemaly/cortex/internal/daemon/config"
)

// SetupGlobal ensures the global ~/.cortex/ directory, settings.yaml, and
// embedded defaults are in place.
func SetupGlobal(force bool) ([]SetupItem, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	cortexDir := filepath.Join(homeDir, ".cortex")
	configPath := filepath.Join(cortexDir, "settings.yaml")

	var items []SetupItem

	item := EnsureDir(cortexDir)
	items = append(items, item)
	if item.Error != nil {
		return items, item.Error
	}

	configContent := `port: 4200
log_level: info
`
	item = EnsureConfigFile(configPath, configContent, force)
	items = append(items, item)
	if item.Error != nil {
		return items, item.Error
	}

	// Auto-populate agent variants if none are configured yet.
	if _, err := EnsureGlobalAgents(configPath); err != nil {
		return items, err
	}

	mainItems, err := setupMainDefaults(homeDir, force)
	if err != nil {
		return append(items, mainItems...), err
	}
	items = append(items, mainItems...)

	return items, nil
}

// RegisterArchitect registers an architect in the daemon's global config.
func RegisterArchitect(projectPath, name string) (bool, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return false, err
	}

	cfg, err := config.Load()
	if err != nil {
		return false, err
	}

	if !cfg.RegisterArchitect(absPath, name) {
		return false, nil
	}

	if err := cfg.Save(); err != nil {
		return false, err
	}
	return true, nil
}

// GlobalAgentsConfigPath returns the path to ~/.cortex/settings.yaml.
func GlobalAgentsConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".cortex", "settings.yaml"), nil
}

// AgentsEmpty reports whether settings.yaml exists but has no agents configured.
func AgentsEmpty(configPath string) bool {
	cfg, err := config.LoadFromFile(configPath)
	if err != nil {
		return false
	}
	return len(cfg.Agents) == 0
}

// EnsureGlobalAgents populates the agents map in settings.yaml if it is empty.
// Returns true if agents were populated, false if already present or no agents detected.
func EnsureGlobalAgents(configPath string) (bool, error) {
	cfg, err := config.LoadFromFile(configPath)
	if err != nil {
		return false, err
	}
	if len(cfg.Agents) > 0 {
		return false, nil
	}

	detected := DetectAgents()
	if !detected.ClaudeAvailable && !detected.OpenCodeAvailable && !detected.CodexAvailable {
		return false, nil
	}

	cfg.Agents = make(map[string]config.AgentVariant)
	if detected.ClaudeAvailable {
		for k, v := range DefaultClaudeVariants() {
			cfg.Agents[k] = v
		}
	}
	if detected.OpenCodeAvailable {
		for k, v := range DefaultOpenCodeVariants() {
			cfg.Agents[k] = v
		}
	}
	if detected.CodexAvailable {
		for k, v := range DefaultCodexVariants() {
			cfg.Agents[k] = v
		}
	}

	return true, cfg.SaveToFile(configPath)
}

func setupMainDefaults(homeDir string, force bool) ([]SetupItem, error) {
	targetDir := filepath.Join(homeDir, ".cortex", "defaults", "main")
	return CopyEmbeddedDefaults("main", targetDir, force)
}

// EnsureDir creates a directory if it doesn't exist.
func EnsureDir(path string) SetupItem {
	item := SetupItem{Path: path}

	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			item.Status = StatusExists
			return item
		}
		item.Error = &PathNotDirectoryError{Path: path}
		return item
	}

	if !os.IsNotExist(err) {
		item.Error = err
		return item
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		item.Error = err
		return item
	}

	item.Status = StatusCreated
	return item
}

// EnsureConfigFile writes a config file if it doesn't exist (or if force is true).
func EnsureConfigFile(path, content string, force bool) SetupItem {
	item := SetupItem{Path: path}

	_, err := os.Stat(path)
	if err == nil {
		if !force {
			item.Status = StatusExists
			return item
		}
	} else if !os.IsNotExist(err) {
		item.Error = err
		return item
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		item.Error = err
		return item
	}

	item.Status = StatusCreated
	return item
}

// PathNotDirectoryError is returned when a path exists but is not a directory.
type PathNotDirectoryError struct {
	Path string
}

func (e *PathNotDirectoryError) Error() string {
	return "path exists but is not a directory: " + e.Path
}

// InstallAgentHooks installs agentstatus hooks for all registered agents.
// Reads daemon port from ~/.cortex/settings.yaml. Non-fatal per-agent errors
// are returned in the result slice; only a systemic failure returns a non-nil error.
func InstallAgentHooks() ([]agentstatus.InstallResult, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config for hook install: %w", err)
	}
	endpoint := fmt.Sprintf("http://localhost:%d/hook", cfg.Port)
	return agentstatus.InstallHooks(agentstatus.InstallConfig{
		Endpoint: endpoint,
		// Agents: nil → all registered adapters (claude, codex, opencode)
	})
}
