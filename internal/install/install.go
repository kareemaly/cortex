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
	// Agent selects which agent defaults to use (claude, opencode). Defaults to claude.
	Agent string
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

	// Create defaults/main directory with prompts
	mainItems, err := setupMainDefaults(homeDir, force)
	if err != nil {
		return append(items, mainItems...), err
	}
	items = append(items, mainItems...)

	return items, nil
}

// setupMainDefaults copies embedded default prompts to ~/.cortex/defaults/main/.
func setupMainDefaults(homeDir string, force bool) ([]SetupItem, error) {
	targetDir := filepath.Join(homeDir, ".cortex", "defaults", "main")
	return CopyEmbeddedDefaults("main", targetDir, force)
}

// setupProject creates the project .cortex/ directory and config.
// Generates a self-contained cortex.yaml with all agent settings inline.
// The extend field points to ~/.cortex/defaults/main for prompt resolution only.
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

	// Generate a self-contained config based on agent type
	configContent := generateProjectConfig(name, agent)
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

// generateProjectConfig returns a complete cortex.yaml for the given agent type.
func generateProjectConfig(name, agent string) string {
	switch agent {
	case "opencode":
		return `name: ` + name + `
extend: ~/.cortex/defaults/main
architect:
  agent: opencode
meta:
  agent: opencode
ticket:
  work:
    agent: opencode
  debug:
    agent: opencode
  research:
    agent: opencode
git:
  worktrees: false
`
	default:
		return `name: ` + name + `
extend: ~/.cortex/defaults/main
architect:
  agent: claude
  args:
    - "--allow-dangerously-skip-permissions"
    - "--allowedTools"
    - "mcp__cortex__listTickets,mcp__cortex__readTicket"
meta:
  agent: claude
  args:
    - "--allow-dangerously-skip-permissions"
    - "--allowedTools"
    - "mcp__cortex__listProjects,mcp__cortex__readProjectConfig,mcp__cortex__readGlobalConfig,mcp__cortex__daemonStatus,mcp__cortex__readDaemonLogs,mcp__cortex__listTickets,mcp__cortex__readTicket,mcp__cortex__listDocs,mcp__cortex__readDoc,mcp__cortex__listSessions,mcp__cortex__readPrompt"
ticket:
  work:
    agent: claude
    args:
      - "--permission-mode"
      - "plan"
      - "--allow-dangerously-skip-permissions"
      - "--allowedTools"
      - "mcp__cortex__readReference"
  debug:
    agent: claude
    args:
      - "--permission-mode"
      - "plan"
      - "--allow-dangerously-skip-permissions"
      - "--allowedTools"
      - "mcp__cortex__readReference,mcp__cortex__addComment"
  research:
    agent: claude
    args:
      - "--allow-dangerously-skip-permissions"
      - "--allowedTools"
      - "mcp__cortex__readReference,mcp__cortex__addComment,mcp__cortex__createDoc"
git:
  worktrees: false
`
	}
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
