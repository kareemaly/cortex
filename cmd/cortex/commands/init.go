package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/kareemaly/cortex/internal/daemon/autostart"
	"github.com/kareemaly/cortex/internal/install"
)

var (
	initGlobalOnly bool
	initForce      bool
	initAgent      string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize project",
	Long: `Initialize Cortex for a project.

Creates the global ~/.cortex/settings.yaml and sets up a project .cortex/
directory in the current directory. Use --global-only to skip project setup.

The agent is auto-detected from your PATH. If both claude and opencode are
available, you'll be prompted to choose. Use --agent to skip detection.`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().BoolVarP(&initGlobalOnly, "global-only", "g", false, "Only set up global ~/.cortex/, skip project setup")
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Overwrite existing config files")
	initCmd.Flags().StringVarP(&initAgent, "agent", "a", "", "Agent type: claude, opencode (auto-detected if not set)")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	agentExplicit := cmd.Flags().Changed("agent")

	// Resolve agent for project setup
	if !initGlobalOnly {
		if agentExplicit {
			// Validate explicit agent flag
			switch initAgent {
			case "claude", "opencode":
				// valid — verify binary exists
				if _, err := exec.LookPath(initAgent); err != nil {
					return fmt.Errorf("%s binary not found in PATH; install it first", initAgent)
				}
			default:
				return fmt.Errorf("invalid agent type %q: must be claude or opencode", initAgent)
			}
		} else {
			// Auto-detect
			agents := install.DetectAgents()
			switch agents.AgentCount() {
			case 0:
				return fmt.Errorf("no supported agent found in PATH\n\nInstall one of:\n  claude  — https://docs.anthropic.com/en/docs/claude-code\n  opencode — https://opencode.ai")
			case 1:
				initAgent = agents.OnlyAgent()
				fmt.Printf("Detected agent: %s\n\n", initAgent)
			case 2:
				isTTY := isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())
				if !isTTY {
					return fmt.Errorf("both claude and opencode found; use --agent to select one")
				}
				selected, err := promptAgentChoice()
				if err != nil {
					return err
				}
				initAgent = selected
			}
		}
	}

	opts := install.Options{
		Agent: initAgent,
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

// promptAgentChoice presents an interactive numbered menu for agent selection.
func promptAgentChoice() (string, error) {
	fmt.Println("Multiple agents detected. Select one:")
	fmt.Println("  1) claude")
	fmt.Println("  2) opencode")
	fmt.Print("Choice [1-2]: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}
	input = strings.TrimSpace(input)

	switch input {
	case "1", "claude":
		fmt.Println()
		return "claude", nil
	case "2", "opencode":
		fmt.Println()
		return "opencode", nil
	default:
		return "", fmt.Errorf("invalid choice %q: enter 1 or 2", input)
	}
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
