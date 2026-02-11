package install

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/kareemaly/cortex/internal/daemon/config"
	projectconfig "github.com/kareemaly/cortex/internal/project/config"
)

// Options configures the installation.
type Options struct {
	// ProjectPath is the path for project setup. If empty, project setup is skipped.
	ProjectPath string
	// ProjectName overrides auto-detected project name.
	ProjectName string
	// Agent selects which agent defaults to use (claude, opencode, copilot). Defaults to claude.
	Agent string
	// Force overwrites existing config files.
	Force bool
}

// DefaultsDirForAgent returns the defaults directory name for the given agent type.
func DefaultsDirForAgent(agent string) string {
	switch agent {
	case "opencode":
		return "opencode"
	case "copilot":
		return "copilot"
	default:
		return "claude-code"
	}
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

		projectItems, err := setupProject(opts.ProjectPath, name, opts.Agent, opts.Force)
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
	// Detect git_diff_tool: use lazygit if available, otherwise git
	gitDiffTool := "git"
	if _, err := exec.LookPath("lazygit"); err == nil {
		gitDiffTool = "lazygit"
	}
	configContent := `port: 4200
log_level: info
status_history_limit: 10
git_diff_tool: ` + gitDiffTool + `
`
	item = ensureConfigFile(configPath, configContent, force)
	items = append(items, item)
	if item.Error != nil {
		return items, item.Error
	}

	// Create defaults/claude-code directory with full config and prompts
	claudeItems, err := setupClaudeCodeDefaults(homeDir, force)
	if err != nil {
		return append(items, claudeItems...), err
	}
	items = append(items, claudeItems...)

	// Create defaults/copilot directory with full config and prompts
	copilotItems, err := setupCopilotDefaults(homeDir, force)
	if err != nil {
		return append(items, copilotItems...), err
	}
	items = append(items, copilotItems...)

	// Create defaults/opencode directory with full config and prompts
	opencodeItems, err := setupOpenCodeDefaults(homeDir, force)
	if err != nil {
		return append(items, opencodeItems...), err
	}
	items = append(items, opencodeItems...)

	return items, nil
}

// setupClaudeCodeDefaults copies embedded default config to ~/.cortex/defaults/claude-code/.
func setupClaudeCodeDefaults(homeDir string, force bool) ([]SetupItem, error) {
	targetDir := filepath.Join(homeDir, ".cortex", "defaults", "claude-code")
	return CopyEmbeddedDefaults("claude-code", targetDir, force)
}

// setupCopilotDefaults copies embedded default config to ~/.cortex/defaults/copilot/.
func setupCopilotDefaults(homeDir string, force bool) ([]SetupItem, error) {
	targetDir := filepath.Join(homeDir, ".cortex", "defaults", "copilot")
	return CopyEmbeddedDefaults("copilot", targetDir, force)
}

// setupOpenCodeDefaults copies embedded default config to ~/.cortex/defaults/opencode/.
func setupOpenCodeDefaults(homeDir string, force bool) ([]SetupItem, error) {
	targetDir := filepath.Join(homeDir, ".cortex", "defaults", "opencode")
	return CopyEmbeddedDefaults("opencode", targetDir, force)
}

// setupProject creates the project .cortex/ directory and config.
// Creates a minimal config that extends the appropriate defaults directory.
// Prompts are inherited from the base config, not copied to the project.
func setupProject(projectPath, name, agent string, force bool) ([]SetupItem, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, err
	}

	cortexDir := filepath.Join(absPath, ".cortex")
	configPath := filepath.Join(cortexDir, "cortex.yaml")

	var items []SetupItem

	// Create .cortex/ directory first
	item := ensureDir(cortexDir)
	items = append(items, item)
	if item.Error != nil {
		return items, item.Error
	}

	// Create minimal config file that extends the appropriate defaults
	// Config must be written before resolving paths so Load() can read it.
	defaultsDir := DefaultsDirForAgent(agent)
	configContent := `name: ` + name + `
extend: ~/.cortex/defaults/` + defaultsDir + `
`
	item = ensureConfigFile(configPath, configContent, force)
	items = append(items, item)
	if item.Error != nil {
		return items, item.Error
	}

	// Load config to resolve tickets/docs paths (respects custom paths from cortex.yaml)
	cfg, err := projectconfig.Load(absPath)
	if err != nil {
		cfg = projectconfig.DefaultConfig()
	}

	// Create tickets directory with status subdirs at config-resolved path
	ticketsDir := cfg.TicketsPath(absPath)
	ticketDirs := []string{
		ticketsDir,
		filepath.Join(ticketsDir, "backlog"),
		filepath.Join(ticketsDir, "progress"),
		filepath.Join(ticketsDir, "review"),
		filepath.Join(ticketsDir, "done"),
	}
	for _, dir := range ticketDirs {
		item := ensureDir(dir)
		items = append(items, item)
		if item.Error != nil {
			return items, item.Error
		}
	}

	// Create docs directory at config-resolved path
	docsDir := cfg.DocsPath(absPath)
	item = ensureDir(docsDir)
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
