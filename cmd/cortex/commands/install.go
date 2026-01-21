package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kareemaly/cortex/internal/install"
)

var (
	installProject bool
	installForce   bool
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Run initial setup",
	Long: `Run initial setup for Cortex.

Creates the global ~/.cortex/ configuration directory and settings file.
Use --project to also set up a project .cortex/ directory in the current directory.`,
	RunE: runInstall,
}

func init() {
	installCmd.Flags().BoolVarP(&installProject, "project", "p", false, "Also set up project .cortex/ in current directory")
	installCmd.Flags().BoolVarP(&installForce, "force", "f", false, "Overwrite existing config files")
	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	opts := install.Options{
		Force: installForce,
	}

	if installProject {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		opts.ProjectPath = cwd
	}

	result, err := install.Run(opts)
	if err != nil {
		return err
	}

	// Print global setup results
	fmt.Println("Global setup:")
	printItems(result.GlobalItems)

	// Print project setup results if applicable
	if len(result.ProjectItems) > 0 {
		fmt.Printf("\nProject setup (%s):\n", result.ProjectName)
		printItems(result.ProjectItems)
	}

	// Print dependencies
	fmt.Println("\nDependencies:")
	for _, dep := range result.Dependencies {
		if dep.Available {
			fmt.Printf("  %s %s installed\n", checkMark(), dep.Name)
		} else {
			fmt.Printf("  %s %s not found (warning)\n", crossMark(), dep.Name)
		}
	}

	return nil
}

func printItems(items []install.SetupItem) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "" // Acceptable fallback for display purposes only
	}

	for _, item := range items {
		path := item.Path
		// Replace home directory with ~
		if homeDir != "" && strings.HasPrefix(path, homeDir) {
			path = "~" + path[len(homeDir):]
		}

		switch item.Status {
		case install.StatusCreated:
			fmt.Printf("  %s Created %s\n", checkMark(), path)
		case install.StatusExists:
			fmt.Printf("  %s %s already exists\n", bullet(), path)
		case install.StatusSkipped:
			fmt.Printf("  - Skipped %s\n", path)
		}

		if item.Error != nil {
			fmt.Printf("    Error: %v\n", item.Error)
		}
	}
}

func checkMark() string {
	return "\u2713"
}

func crossMark() string {
	return "\u2717"
}

func bullet() string {
	return "\u2022"
}
