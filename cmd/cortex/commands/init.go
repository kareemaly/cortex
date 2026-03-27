package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kareemaly/cortex/internal/install"
	"github.com/spf13/cobra"
)

var (
	initForce bool
	initAgent string
)

var initCmd = &cobra.Command{
	Use:   "init <name>",
	Short: "Initialize a new architect workspace",
	Long: `Initialize a new Cortex architect workspace.

Creates the global ~/.cortex/ directory if needed, scaffolds a project directory
with ticket storage, generates cortex.yaml with agent variants, and registers the
architect with the daemon.

The agent is auto-detected from your PATH (prefers claude). Use --agent to override.`,
	Args: cobra.ExactArgs(1),
	RunE: runInit,
}

func init() {
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Overwrite existing config files")
	initCmd.Flags().StringVarP(&initAgent, "agent", "a", "", "Agent type: claude, opencode (auto-detected if not set)")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	name := args[0]

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	projectPath := filepath.Join(cwd, name)

	// Ensure global ~/.cortex/ setup
	globalItems, err := install.SetupGlobal(initForce)
	if err != nil {
		return fmt.Errorf("global setup failed: %w", err)
	}

	// Detect agent
	agent := initAgent
	if agent == "" {
		detected := install.DetectAgents()
		if detected.ClaudeAvailable {
			agent = "claude"
		} else if detected.OpenCodeAvailable {
			agent = "opencode"
		} else {
			return fmt.Errorf("no supported agent found in PATH (install claude or opencode)")
		}
	}
	if agent != "claude" && agent != "opencode" {
		return fmt.Errorf("unsupported agent %q (must be claude or opencode)", agent)
	}

	// Create project directories
	dirs := []string{
		projectPath,
		filepath.Join(projectPath, "tickets"),
		filepath.Join(projectPath, "tickets", "backlog"),
		filepath.Join(projectPath, "tickets", "progress"),
		filepath.Join(projectPath, "tickets", "done"),
		filepath.Join(projectPath, "sessions"),
	}
	for _, dir := range dirs {
		item := install.EnsureDir(dir)
		if item.Error != nil {
			return fmt.Errorf("failed to create %s: %w", dir, item.Error)
		}
	}

	// Generate cortex.yaml
	configContent := generateConfig(name, agent)
	configPath := filepath.Join(projectPath, "cortex.yaml")
	item := install.EnsureConfigFile(configPath, configContent, initForce)
	if item.Error != nil {
		return fmt.Errorf("failed to write cortex.yaml: %w", item.Error)
	}

	// Register with daemon
	registered, regErr := install.RegisterArchitect(projectPath, name)

	// Print summary
	fmt.Println()
	for _, gi := range globalItems {
		printItem(gi)
	}
	for _, dir := range dirs {
		fmt.Printf("  ✓ %s\n", dir)
	}
	printItem(item)
	fmt.Println()

	if registered {
		fmt.Printf("  Registered architect %q with daemon\n", name)
	} else if regErr != nil {
		fmt.Printf("  Warning: failed to register architect: %v\n", regErr)
	}

	fmt.Printf("\nArchitect %q initialized with %s agent.\n", name, agent)
	fmt.Println("\nNext steps:")
	fmt.Printf("  1. Edit %s to add your repos\n", configPath)
	fmt.Printf("  2. Run: cd %s && cortex architect start\n", name)
	fmt.Println()

	return nil
}

func printItem(item install.SetupItem) {
	switch item.Status {
	case install.StatusCreated:
		fmt.Printf("  + %s\n", item.Path)
	case install.StatusExists:
		fmt.Printf("  · %s (exists)\n", item.Path)
	}
}

func generateConfig(name, agent string) string {
	var sb strings.Builder

	sb.WriteString("name: ")
	sb.WriteString(name)
	sb.WriteString("\nrepos: []\n")

	sb.WriteString("\nagents:\n")

	sb.WriteString("  ")
	sb.WriteString(agent)
	sb.WriteString(":\n")
	sb.WriteString("    agent: ")
	sb.WriteString(agent)
	sb.WriteString("\n    args: []\n")

	sb.WriteString("  ")
	sb.WriteString(agent)
	sb.WriteString("-plan:\n")
	sb.WriteString("    agent: ")
	sb.WriteString(agent)
	sb.WriteString("\n")

	if agent == "claude" {
		sb.WriteString("    args: [\"--permission-mode\", \"plan\"]\n")
	} else {
		sb.WriteString("    args: [\"--agent\", \"plan\"]\n")
	}

	return sb.String()
}
