package commands

import (
	"errors"
	"fmt"
	"os"

	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/tmux"
	"github.com/spf13/cobra"
)

var (
	metaDetachFlag bool
	metaModeFlag   string
)

var metaCmd = &cobra.Command{
	Use:   "meta",
	Short: "Start or attach to meta session",
	Long: `Starts or attaches to the global meta session.

The meta agent operates above projects — it can register projects, spawn
architects, configure workflows, customize prompts, and debug daemon issues.

If a meta session already exists, it will be focused.
Otherwise, a fresh meta session will be spawned.

If a session was orphaned (tmux window closed but session record remains),
use --mode fresh to start a new session or --mode resume to continue.

Examples:
  cortex meta              # Start or attach to meta session
  cortex meta --detach     # Start meta without attaching
  cortex meta --mode fresh # Clear orphaned session, start new`,
	Run: runMeta,
}

func init() {
	metaCmd.Flags().BoolVar(&metaDetachFlag, "detach", false,
		"Spawn meta without attaching to session")
	metaCmd.Flags().StringVar(&metaModeFlag, "mode", "",
		"Spawn mode: normal (default), fresh (clear orphaned session), resume (continue orphaned session)")
	rootCmd.AddCommand(metaCmd)
}

func runMeta(cmd *cobra.Command, args []string) {
	ensureDaemon()

	// Create SDK client (no project path needed for meta)
	client := sdk.DefaultClient("")

	// Spawn meta via SDK
	resp, err := client.SpawnMeta(metaModeFlag)
	if err != nil {
		var apiErr *sdk.APIError
		if errors.As(err, &apiErr) && apiErr.IsOrphanedSession() {
			fmt.Fprintf(os.Stderr, "Meta session is orphaned (tmux window was closed but session record remains).\n\n")
			fmt.Fprintf(os.Stderr, "Options:\n")
			fmt.Fprintf(os.Stderr, "  cortex meta --mode fresh   # Start a new session\n")
			fmt.Fprintf(os.Stderr, "  cortex meta --mode resume  # Resume the previous session\n")
			os.Exit(1)
		}
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
