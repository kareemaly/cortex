package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	architectconfig "github.com/kareemaly/cortex/internal/architect/config"
	"github.com/kareemaly/cortex/internal/daemon/config"
)

type Options struct {
	ArchitectPath   string
	ArchitectName   string
	Agent           string
	Model           string
	Force           bool
	Repos           []string
	Companion       string
	FeatureBranches bool
	ResearchPaths   []string
}

func Run(opts Options) (*Result, error) {
	result := &Result{}

	globalItems, err := setupGlobal(opts.Force)
	if err != nil {
		return nil, err
	}
	result.GlobalItems = globalItems

	if opts.ArchitectPath != "" {
		name := opts.ArchitectName
		if name == "" {
			name = DetectArchitectName(opts.ArchitectPath)
		}
		result.ArchitectName = name

		projectItems, err := setupProject(opts, name)
		if err != nil {
			return nil, err
		}
		result.ArchitectItems = projectItems

		registered, regErr := registerArchitect(opts.ArchitectPath, name)
		result.Registered = registered
		result.RegistrationError = regErr
	}

	result.Dependencies = CheckDependencies()

	return result, nil
}

func registerArchitect(projectPath, name string) (bool, error) {
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

func setupGlobal(force bool) ([]SetupItem, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	cortexDir := filepath.Join(homeDir, ".cortex")
	configPath := filepath.Join(cortexDir, "settings.yaml")

	var items []SetupItem

	item := ensureDir(cortexDir)
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
	item = ensureConfigFile(configPath, configContent, force)
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

func setupMainDefaults(homeDir string, force bool) ([]SetupItem, error) {
	targetDir := filepath.Join(homeDir, ".cortex", "defaults", "main")
	return CopyEmbeddedDefaults("main", targetDir, force)
}

func setupProject(opts Options, name string) ([]SetupItem, error) {
	absPath, err := filepath.Abs(opts.ArchitectPath)
	if err != nil {
		return nil, err
	}

	var items []SetupItem

	item := ensureDir(absPath)
	items = append(items, item)
	if item.Error != nil {
		return items, item.Error
	}

	configPath := filepath.Join(absPath, "cortex.yaml")
	configContent := generateProjectConfig(opts, name)
	item = ensureConfigFile(configPath, configContent, opts.Force)
	items = append(items, item)
	if item.Error != nil {
		return items, item.Error
	}

	cfg, err := architectconfig.Load(absPath)
	if err != nil {
		cfg = architectconfig.DefaultConfig()
	}

	ticketsDir := cfg.TicketsPath(absPath)
	ticketDirs := []string{
		ticketsDir,
		filepath.Join(ticketsDir, "backlog"),
		filepath.Join(ticketsDir, "progress"),
		filepath.Join(ticketsDir, "done"),
	}
	for _, dir := range ticketDirs {
		item := ensureDir(dir)
		items = append(items, item)
		if item.Error != nil {
			return items, item.Error
		}
	}

	promptItems, err := setupProjectPrompts(absPath, opts)
	if err != nil {
		return items, err
	}
	items = append(items, promptItems...)

	return items, nil
}

func setupProjectPrompts(architectRoot string, opts Options) ([]SetupItem, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	srcDir := filepath.Join(homeDir, ".cortex", "defaults", "main", "prompts")
	dstDir := filepath.Join(architectRoot, "prompts")

	var items []SetupItem

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read prompts directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		roleDir := entry.Name()
		srcRoleDir := filepath.Join(srcDir, roleDir)
		dstRoleDir := filepath.Join(dstDir, roleDir)

		item := ensureDir(dstRoleDir)
		items = append(items, item)
		if item.Error != nil {
			return items, item.Error
		}

		files, err := os.ReadDir(srcRoleDir)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s prompts: %w", roleDir, err)
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			srcFile := filepath.Join(srcRoleDir, file.Name())
			dstFile := filepath.Join(dstRoleDir, file.Name())

			content, err := os.ReadFile(srcFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read %s: %w", srcFile, err)
			}

			if roleDir == "work" && file.Name() == "KICKOFF.md" && opts.FeatureBranches {
				content = append(content, []byte(featureBranchSection)...)
			}

			item := ensureConfigFile(dstFile, string(content), opts.Force)
			items = append(items, item)
			if item.Error != nil {
				return items, item.Error
			}
		}
	}

	return items, nil
}

const featureBranchSection = `

## Workflow

Start by ensuring you're on a clean git state, then create a feature branch:
` + "`git checkout -b ticket/{ticket-id}-{slug}`" + `

If the working directory is not clean, ask the user how to proceed.
`

func generateProjectConfig(opts Options, name string) string {
	var sb strings.Builder

	sb.WriteString("name: ")
	sb.WriteString(name)
	sb.WriteString("\n")

	if len(opts.Repos) > 0 {
		sb.WriteString("\nrepos:\n")
		for _, repo := range opts.Repos {
			sb.WriteString("  - ")
			sb.WriteString(repo)
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\narchitect:\n")
	sb.WriteString("  agent: ")
	sb.WriteString(opts.Agent)
	sb.WriteString("\n")
	if opts.Model != "" {
		sb.WriteString("  args: [\"--model\", \"")
		sb.WriteString(opts.Model)
		sb.WriteString("\"]\n")
	}

	sb.WriteString("\nwork:\n")
	sb.WriteString("  agent: ")
	sb.WriteString(opts.Agent)
	sb.WriteString("\n")
	if opts.Model != "" {
		sb.WriteString("  args: [\"--model\", \"")
		sb.WriteString(opts.Model)
		sb.WriteString("\"]\n")
	}
	if opts.Companion != "" {
		sb.WriteString("  companion: ")
		sb.WriteString(opts.Companion)
		sb.WriteString("\n")
	}

	sb.WriteString("\nresearch:\n")
	sb.WriteString("  agent: ")
	sb.WriteString(opts.Agent)
	sb.WriteString("\n")
	if opts.Model != "" {
		sb.WriteString("  args: [\"--model\", \"")
		sb.WriteString(opts.Model)
		sb.WriteString("\"]\n")
	}
	if len(opts.ResearchPaths) > 0 {
		sb.WriteString("  paths:\n")
		for _, path := range opts.ResearchPaths {
			sb.WriteString("    - ")
			sb.WriteString(path)
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\ncollab:\n")
	sb.WriteString("  agent: ")
	sb.WriteString(opts.Agent)
	sb.WriteString("\n")
	if opts.Model != "" {
		sb.WriteString("  args: [\"--model\", \"")
		sb.WriteString(opts.Model)
		sb.WriteString("\"]\n")
	}
	if opts.Companion != "" {
		sb.WriteString("  companion: ")
		sb.WriteString(opts.Companion)
		sb.WriteString("\n")
	}

	return sb.String()
}

func ensureDir(path string) SetupItem {
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

func ensureConfigFile(path, content string, force bool) SetupItem {
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

type PathNotDirectoryError struct {
	Path string
}

func (e *PathNotDirectoryError) Error() string {
	return "path exists but is not a directory: " + e.Path
}
