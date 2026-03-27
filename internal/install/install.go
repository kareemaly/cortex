package install

import (
	"os"
	"os/exec"
	"path/filepath"

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

	gitDiffTool := "git"
	if _, err := exec.LookPath("lazygit"); err == nil {
		gitDiffTool = "lazygit"
	}
	configContent := `port: 4200
log_level: info
status_history_limit: 10
git_diff_tool: ` + gitDiffTool + `
`
	item = EnsureConfigFile(configPath, configContent, force)
	items = append(items, item)
	if item.Error != nil {
		return items, item.Error
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
