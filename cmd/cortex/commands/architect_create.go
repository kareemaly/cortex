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
	architectCreateGlobalOnly bool
	architectCreateForce      bool
	architectCreateAgent      string
)

var architectCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Initialize a new architect workspace",
	Long: `Initialize Cortex for an architect workspace.

Creates the global ~/.cortex/settings.yaml and sets up a cortex.yaml
in the current directory. Use --global-only to skip project setup.

The agent is auto-detected from your PATH. If both claude and opencode are
available, you'll be prompted to choose. Use --agent to skip detection.`,
	RunE: runArchitectCreate,
}

func init() {
	architectCreateCmd.Flags().BoolVarP(&architectCreateGlobalOnly, "global-only", "g", false, "Only set up global ~/.cortex/, skip project setup")
	architectCreateCmd.Flags().BoolVarP(&architectCreateForce, "force", "f", false, "Overwrite existing config files")
	architectCreateCmd.Flags().StringVarP(&architectCreateAgent, "agent", "a", "", "Agent type: claude, opencode (auto-detected if not set)")
	architectCmd.AddCommand(architectCreateCmd)
}

func runArchitectCreate(cmd *cobra.Command, args []string) error {
	agentExplicit := cmd.Flags().Changed("agent")
	agent := architectCreateAgent

	// Resolve agent for project setup
	if !architectCreateGlobalOnly {
		if agentExplicit {
			// Validate explicit agent flag
			switch agent {
			case "claude", "opencode":
				// valid — verify binary exists
				if _, err := exec.LookPath(agent); err != nil {
					return fmt.Errorf("%s binary not found in PATH; install it first", agent)
				}
			default:
				return fmt.Errorf("invalid agent type %q: must be claude or opencode", agent)
			}
		} else {
			// Auto-detect
			agents := install.DetectAgents()
			switch agents.AgentCount() {
			case 0:
				return fmt.Errorf("no supported agent found in PATH\n\nInstall one of:\n  claude  — https://docs.anthropic.com/en/docs/claude-code\n  opencode — https://opencode.ai")
			case 1:
				agent = agents.OnlyAgent()
				fmt.Printf("Detected agent: %s\n\n", agent)
			case 2:
				isTTY := isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())
				if !isTTY {
					return fmt.Errorf("both claude and opencode found; use --agent to select one")
				}
				selected, err := architectPromptAgentChoice()
				if err != nil {
					return err
				}
				agent = selected
			}
		}
	}

	opts := install.Options{
		Agent: agent,
		Force: architectCreateForce,
	}

	if !architectCreateGlobalOnly {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		opts.ArchitectPath = cwd
	}

	result, err := install.Run(opts)
	if err != nil {
		return err
	}

	// Print global setup results
	fmt.Println("Global setup:")
	printItems(result.GlobalItems)

	// Print project setup results if applicable
	if len(result.ArchitectItems) > 0 {
		fmt.Printf("\nProject setup (%s):\n", result.ArchitectName)
		printItems(result.ArchitectItems)

		// Print registration status
		fmt.Println("\nGlobal registry:")
		if result.RegistrationError != nil {
			fmt.Printf("  %s Failed to register architect: %v\n", crossMark(), result.RegistrationError)
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

// architectPromptAgentChoice presents an interactive numbered menu for agent selection.
func architectPromptAgentChoice() (string, error) {
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
