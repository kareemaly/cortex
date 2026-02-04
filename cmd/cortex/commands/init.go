package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kareemaly/cortex/internal/daemon/autostart"
	"github.com/kareemaly/cortex/internal/install"
)

var (
	initGlobalOnly bool
	initForce      bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize project",
	Long: `Initialize Cortex for a project.

Creates the global ~/.cortex/settings.yaml and sets up a project .cortex/
directory in the current directory. Use --global-only to skip project setup.`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().BoolVarP(&initGlobalOnly, "global-only", "g", false, "Only set up global ~/.cortex/, skip project setup")
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Overwrite existing config files")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	opts := install.Options{
		Force: initForce,
	}

	if !initGlobalOnly {
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

		// Print registration status
		fmt.Println("\nGlobal registry:")
		if result.RegistrationError != nil {
			fmt.Printf("  %s Failed to register project: %v\n", crossMark(), result.RegistrationError)
		} else if result.Registered {
			fmt.Printf("  %s Registered in ~/.cortex/settings.yaml\n", checkMark())
		} else {
			fmt.Printf("  %s Already registered in ~/.cortex/settings.yaml\n", bullet())
		}
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

	// Start daemon
	fmt.Println("\nDaemon:")
	if err := autostart.EnsureDaemonRunning(); err != nil {
		fmt.Printf("  %s Failed to start daemon: %v\n", crossMark(), err)
		fmt.Println("    Run 'cortex daemon restart' to try again")
	} else {
		fmt.Printf("  %s Daemon running\n", checkMark())
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
