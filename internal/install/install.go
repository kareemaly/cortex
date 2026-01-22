package install

import (
	"os"
	"path/filepath"

	"github.com/kareemaly/cortex/internal/prompt"
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
	}

	// Check dependencies
	result.Dependencies = CheckDependencies()

	return result, nil
}

// setupGlobal creates the global ~/.cortex/ directory and config.
func setupGlobal(force bool) ([]SetupItem, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	cortexDir := filepath.Join(homeDir, ".cortex")
	ticketsDir := filepath.Join(cortexDir, "tickets")
	backlogDir := filepath.Join(ticketsDir, "backlog")
	progressDir := filepath.Join(ticketsDir, "progress")
	doneDir := filepath.Join(ticketsDir, "done")
	configPath := filepath.Join(cortexDir, "settings.yaml")

	var items []SetupItem

	// Create directories
	dirs := []string{cortexDir, ticketsDir, backlogDir, progressDir, doneDir}
	for _, dir := range dirs {
		item := ensureDir(dir)
		items = append(items, item)
		if item.Error != nil {
			return items, item.Error
		}
	}

	// Create config file
	configContent := `port: 4200
log_level: info
status_history_limit: 10
git_diff_tool: diff
`
	item := ensureConfigFile(configPath, configContent, force)
	items = append(items, item)
	if item.Error != nil {
		return items, item.Error
	}

	return items, nil
}

// setupProject creates the project .cortex/ directory and config.
func setupProject(projectPath, name string, force bool) ([]SetupItem, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, err
	}

	cortexDir := filepath.Join(absPath, ".cortex")
	ticketsDir := filepath.Join(cortexDir, "tickets")
	backlogDir := filepath.Join(ticketsDir, "backlog")
	progressDir := filepath.Join(ticketsDir, "progress")
	doneDir := filepath.Join(ticketsDir, "done")
	configPath := filepath.Join(cortexDir, "cortex.yaml")

	var items []SetupItem

	// Create directories
	dirs := []string{cortexDir, ticketsDir, backlogDir, progressDir, doneDir}
	for _, dir := range dirs {
		item := ensureDir(dir)
		items = append(items, item)
		if item.Error != nil {
			return items, item.Error
		}
	}

	// Create config file
	configContent := `name: ` + name + `
agent: claude
git:
  repos:
    - path: "."
`
	item := ensureConfigFile(configPath, configContent, force)
	items = append(items, item)
	if item.Error != nil {
		return items, item.Error
	}

	// Create prompts directory and default templates
	promptsDir := prompt.PromptsDir(absPath)
	item = ensureDir(promptsDir)
	items = append(items, item)
	if item.Error != nil {
		return items, item.Error
	}

	architectPath := prompt.ArchitectPath(absPath)
	item = ensureConfigFile(architectPath, prompt.DefaultArchitectPrompt, force)
	items = append(items, item)
	if item.Error != nil {
		return items, item.Error
	}

	ticketAgentPath := prompt.TicketAgentPath(absPath)
	item = ensureConfigFile(ticketAgentPath, prompt.DefaultTicketAgentPrompt, force)
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
