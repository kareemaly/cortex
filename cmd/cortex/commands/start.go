package commands

import (
	"fmt"
	"os"

	"github.com/kareemaly/cortex/internal/tmux"
	"github.com/spf13/cobra"
)

const daemonSessionName = "CortexDaemon"

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start or attach to the Cortex daemon session",
	Long: `Creates or attaches to the CortexDaemon tmux session.

The session has two panes:
  - Left (70%):  cortex dashboard
  - Right (30%): cortexd (daemon logs)

If the session already exists, it will be attached/switched to.

Examples:
  cortex start   # Create or attach to CortexDaemon session`,
	Run: runStart,
}

func init() {
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) {
	manager, err := tmux.NewManager()
	if err != nil {
		if tmux.IsNotInstalled(err) {
			fmt.Fprintf(os.Stderr, "Error: tmux is required but not installed\n")
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	exists, err := manager.SessionExists(daemonSessionName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if exists {
		if err := manager.AttachSession(daemonSessionName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to attach to session: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Create new session.
	if err := manager.CreateSession(daemonSessionName, ""); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create session: %v\n", err)
		os.Exit(1)
	}

	// Split: right pane = 30% for daemon.
	if err := manager.SplitWindowHorizontalWithPercent(daemonSessionName, 0, 30, ""); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to split window: %v\n", err)
		os.Exit(1)
	}

	// Run daemon in right pane (pane 1).
	if err := manager.RunCommandInPane(daemonSessionName, 0, 1, "cortexd"); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to start daemon: %v\n", err)
		os.Exit(1)
	}

	// Run dashboard in left pane (pane 0).
	if err := manager.RunCommandInPane(daemonSessionName, 0, 0, "cortex dashboard"); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to start dashboard: %v\n", err)
		os.Exit(1)
	}

	// Focus dashboard pane.
	if err := manager.SelectPane(daemonSessionName, 0, 0); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to select pane: %v\n", err)
		os.Exit(1)
	}

	// Attach to the session.
	if err := manager.AttachSession(daemonSessionName); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to attach to session: %v\n", err)
		os.Exit(1)
	}
}
