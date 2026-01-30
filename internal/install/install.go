package install

import (
	"os"
	"path/filepath"

	"github.com/kareemaly/cortex/internal/daemon/config"
)

// Options configures the installation.
type Options struct {
	// ProjectPath is the path for project setup. If empty, project setup is skipped.
	ProjectPath string
	// ProjectName overrides auto-detected project name.
	ProjectName string
	// Force overwrites existing config files.
	Force bool
}

// Run performs the installation with the given options.
func Run(opts Options) (*Result, error) {
	result := &Result{}

	// Always setup global config
	globalItems, err := setupGlobal(opts.Force)
	if err != nil {
		return nil, err
	}
	result.GlobalItems = globalItems

	// Setup project if path is provided
	if opts.ProjectPath != "" {
		name := opts.ProjectName
		if name == "" {
			name = DetectProjectName(opts.ProjectPath)
		}
		result.ProjectName = name

		projectItems, err := setupProject(opts.ProjectPath, name, opts.Force)
		if err != nil {
			return nil, err
		}
		result.ProjectItems = projectItems

		// Auto-register project in global config (non-fatal)
		registered, regErr := registerProject(opts.ProjectPath, name)
		result.Registered = registered
		result.RegistrationError = regErr
	}

	// Check dependencies
	result.Dependencies = CheckDependencies()

	return result, nil
}

// registerProject adds the project to the global settings.yaml registry.
func registerProject(projectPath, name string) (bool, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return false, err
	}

	cfg, err := config.Load()
	if err != nil {
		return false, err
	}

	if !cfg.RegisterProject(absPath, name) {
		return false, nil // already registered
	}

	if err := cfg.Save(); err != nil {
		return false, err
	}
	return true, nil
}

// setupGlobal creates the global ~/.cortex/ directory and config.
func setupGlobal(force bool) ([]SetupItem, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	cortexDir := filepath.Join(homeDir, ".cortex")
	configPath := filepath.Join(cortexDir, "settings.yaml")

	var items []SetupItem

	// Create cortex directory
	item := ensureDir(cortexDir)
	items = append(items, item)
	if item.Error != nil {
		return items, item.Error
	}

	// Create config file
	configContent := `port: 4200
log_level: info
status_history_limit: 10
git_diff_tool: diff
`
	item = ensureConfigFile(configPath, configContent, force)
	items = append(items, item)
	if item.Error != nil {
		return items, item.Error
	}

	// Create defaults/claude-code directory with full config and prompts
	basicItems, err := setupClaudeCodeDefaults(homeDir, force)
	if err != nil {
		return append(items, basicItems...), err
	}
	items = append(items, basicItems...)

	return items, nil
}

// setupClaudeCodeDefaults copies embedded default config to ~/.cortex/defaults/claude-code/.
func setupClaudeCodeDefaults(homeDir string, force bool) ([]SetupItem, error) {
	targetDir := filepath.Join(homeDir, ".cortex", "defaults", "claude-code")
	return copyEmbeddedDefaults("claude-code", targetDir, force)
}

// setupProject creates the project .cortex/ directory and config.
// Creates a minimal config that extends ~/.cortex/defaults/claude-code.
// Prompts are inherited from the base config, not copied to the project.
func setupProject(projectPath, name string, force bool) ([]SetupItem, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, err
	}

	cortexDir := filepath.Join(absPath, ".cortex")
	ticketsDir := filepath.Join(cortexDir, "tickets")
	backlogDir := filepath.Join(ticketsDir, "backlog")
	progressDir := filepath.Join(ticketsDir, "progress")
	reviewDir := filepath.Join(ticketsDir, "review")
	doneDir := filepath.Join(ticketsDir, "done")
	configPath := filepath.Join(cortexDir, "cortex.yaml")

	var items []SetupItem

	// Create directories (no prompts directory - inherited from base)
	dirs := []string{cortexDir, ticketsDir, backlogDir, progressDir, reviewDir, doneDir}
	for _, dir := range dirs {
		item := ensureDir(dir)
		items = append(items, item)
		if item.Error != nil {
			return items, item.Error
		}
	}

	// Create minimal config file that extends the claude-code defaults
	configContent := `name: ` + name + `
extend: ~/.cortex/defaults/claude-code
`
	item := ensureConfigFile(configPath, configContent, force)
	items = append(items, item)
	if item.Error != nil {
		return items, item.Error
	}

	return items, nil
}

// ensureDir creates a directory if it doesn't exist.
func ensureDir(path string) SetupItem {
	item := SetupItem{Path: path}

	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			item.Status = StatusExists
			return item
		}
		// Path exists but is not a directory
		item.Error = &PathNotDirectoryError{Path: path}
		return item
	}

	if !os.IsNotExist(err) {
		item.Error = err
		return item
	}

	// Create the directory
	if err := os.MkdirAll(path, 0755); err != nil {
		item.Error = err
		return item
	}

	item.Status = StatusCreated
	return item
}

// ensureConfigFile creates a config file if it doesn't exist or force is true.
func ensureConfigFile(path, content string, force bool) SetupItem {
	item := SetupItem{Path: path}

	_, err := os.Stat(path)
	if err == nil {
		if !force {
			item.Status = StatusExists
			return item
		}
		// Force overwrite
	} else if !os.IsNotExist(err) {
		item.Error = err
		return item
	}

	// Write the file
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		item.Error = err
		return item
	}

	item.Status = StatusCreated
	return item
}

// PathNotDirectoryError indicates a path exists but is not a directory.
type PathNotDirectoryError struct {
	Path string
}

func (e *PathNotDirectoryError) Error() string {
	return "path exists but is not a directory: " + e.Path
}
