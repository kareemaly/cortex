package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	projectconfig "github.com/kareemaly/cortex/internal/project/config"
)

var (
	configShowPath string
)

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display resolved project configuration",
	Long: `Display the project configuration.

This helps verify your config is correct.
The output is valid YAML that could be copy-pasted into a config file.`,
	RunE: runConfigShow,
}

func init() {
	configShowCmd.Flags().StringVarP(&configShowPath, "path", "p", "", "Path to project (defaults to current directory)")
}

// configOutput represents the config structure for YAML output.
// We use a separate struct to control field ordering and omit internal fields.
type configOutput struct {
	Name      string                      `yaml:"name,omitempty"`
	Architect roleConfigOutput            `yaml:"architect,omitempty"`
	Ticket    map[string]roleConfigOutput `yaml:"ticket,omitempty"`
	Git       gitConfigOutput             `yaml:"git,omitempty"`
}

type roleConfigOutput struct {
	Agent projectconfig.AgentType `yaml:"agent,omitempty"`
	Args  []string                `yaml:"args,omitempty"`
}

type gitConfigOutput struct {
	Worktrees bool `yaml:"worktrees"`
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	// Determine the path to use
	path := configShowPath
	if path == "" {
		var err error
		path, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Find project root and load config
	cfg, projectRoot, err := projectconfig.LoadFromPath(path)
	if err != nil {
		if projectconfig.IsProjectNotFound(err) {
			return fmt.Errorf("no cortex project found at %s\n\nRun 'cortex init' to initialize a project", path)
		}
		if projectconfig.IsExtendPathNotFound(err) {
			return fmt.Errorf("config error: %w", err)
		}
		if projectconfig.IsConfigParseError(err) {
			return fmt.Errorf("config error: %w", err)
		}
		return err
	}

	// Print comment header
	fmt.Printf("# Resolved config for: %s\n", projectRoot)
	if extendPath := cfg.ResolvedExtendPath(); extendPath != "" {
		fmt.Printf("# Extended from: %s\n", formatPath(extendPath))
	}
	fmt.Println()

	// Convert to output structure (omits internal fields like Extend and resolvedExtendPath)
	output := configOutput{
		Name: cfg.Name,
		Architect: roleConfigOutput{
			Agent: cfg.Architect.Agent,
			Args:  cfg.Architect.Args,
		},
		Git: gitConfigOutput{
			Worktrees: cfg.Git.Worktrees,
		},
	}

	// Convert ticket config
	if len(cfg.Ticket) > 0 {
		output.Ticket = make(map[string]roleConfigOutput)
		for k, v := range cfg.Ticket {
			output.Ticket[k] = roleConfigOutput{
				Agent: v.Agent,
				Args:  v.Args,
			}
		}
	}

	// Marshal to YAML
	data, err := yaml.Marshal(output)
	if err != nil {
		return fmt.Errorf("failed to format config: %w", err)
	}

	fmt.Print(string(data))
	return nil
}

// formatPath replaces home directory with ~ for display.
func formatPath(path string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, homeDir) {
		return "~" + path[len(homeDir):]
	}
	return path
}
