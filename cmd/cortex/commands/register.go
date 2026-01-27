package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kareemaly/cortex/internal/daemon/config"
	"github.com/kareemaly/cortex/internal/install"
	"github.com/spf13/cobra"
)

var registerCmd = &cobra.Command{
	Use:   "register [path]",
	Short: "Register a project in the global registry",
	Long: `Register a project in ~/.cortex/settings.yaml.
Defaults to the current directory if no path is provided.
The project must have a .cortex/ directory (run 'cortex init' first).`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		absPath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("failed to resolve path: %w", err)
		}

		// Validate .cortex/ directory exists
		cortexDir := filepath.Join(absPath, ".cortex")
		info, err := os.Stat(cortexDir)
		if err != nil || !info.IsDir() {
			return fmt.Errorf("no .cortex directory found at %s (run 'cortex init' first)", absPath)
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		title := install.DetectProjectName(absPath)
		if !cfg.RegisterProject(absPath, title) {
			fmt.Printf("Project already registered: %s\n", absPath)
			return nil
		}

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("Registered project: %s (%s)\n", title, absPath)
		return nil
	},
}

var unregisterCmd = &cobra.Command{
	Use:   "unregister [path]",
	Short: "Remove a project from the global registry",
	Long: `Remove a project from ~/.cortex/settings.yaml.
Defaults to the current directory if no path is provided.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		absPath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("failed to resolve path: %w", err)
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if !cfg.UnregisterProject(absPath) {
			fmt.Printf("Project not found in registry: %s\n", absPath)
			return nil
		}

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("Unregistered project: %s\n", absPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(registerCmd)
	rootCmd.AddCommand(unregisterCmd)
}
