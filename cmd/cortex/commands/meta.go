package commands

import (
	"fmt"
	"os"

	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/tmux"
	"github.com/spf13/cobra"
)

var metaDetachFlag bool

var metaCmd = &cobra.Command{
	Use:   "meta",
	Short: "Start or attach to meta session",
	Long: `Starts or attaches to the global meta session.

The meta agent operates above projects — it can register projects, spawn
architects, configure workflows, customize prompts, and debug daemon issues.

If a meta session already exists, it will be focused.
Otherwise, a fresh meta session will be spawned. Orphaned sessions are
automatically cleaned up.

Examples:
  cortex meta              # Start or attach to meta session
  cortex meta --detach     # Start meta without attaching`,
	Run: runMeta,
}

func init() {
	metaCmd.Flags().BoolVar(&metaDetachFlag, "detach", false,
		"Spawn meta without attaching to session")
	rootCmd.AddCommand(metaCmd)
}

func runMeta(cmd *cobra.Command, args []string) {
	ensureDaemon()

	// Create SDK client (no project path needed for meta)
	client := sdk.DefaultClient("")

	// Spawn meta via SDK
	resp, err := client.SpawnMeta()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Attach to tmux session unless --detach is set
	if !metaDetachFlag {
		manager, err := tmux.NewManager()
		if err != nil {
			if tmux.IsNotInstalled(err) {
				fmt.Fprintf(os.Stderr, "Error: tmux is required but not installed\n")
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := manager.AttachSession(resp.TmuxSession); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to attach to session: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Printf("Meta running in tmux session: %s\n", resp.TmuxSession)
	}
}
