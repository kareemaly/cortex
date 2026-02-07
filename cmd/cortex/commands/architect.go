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
	architectDetachFlag bool
	architectModeFlag   string
)

var architectCmd = &cobra.Command{
	Use:   "architect",
	Short: "Start or attach to architect session",
	Long: `Starts or attaches to the architect session for this project.

The architect lives in tmux window 0 and has full MCP tools for ticket
management and spawning ticket sessions.

If an architect window already exists, it will be focused.
Otherwise, a fresh architect session will be spawned.

If a session was orphaned (tmux window closed but session record remains),
use --mode fresh to start a new session or --mode resume to continue.

Examples:
  cortex architect              # Start or attach to architect
  cortex architect --detach     # Start architect without attaching
  cortex architect --mode fresh # Clear orphaned session, start new`,
	Run: runArchitect,
}

func init() {
	architectCmd.Flags().BoolVar(&architectDetachFlag, "detach", false,
		"Spawn architect without attaching to session")
	architectCmd.Flags().StringVar(&architectModeFlag, "mode", "",
		"Spawn mode: normal (default), fresh (clear orphaned session), resume (continue orphaned session)")
	rootCmd.AddCommand(architectCmd)
}

func runArchitect(cmd *cobra.Command, args []string) {
	ensureDaemon()

	// Get project path
	projectPath, err := resolveProjectPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Create SDK client
	client := sdk.DefaultClient(projectPath)

	// Spawn architect via SDK
	resp, err := client.SpawnArchitect(architectModeFlag)
	if err != nil {
		var apiErr *sdk.APIError
		if errors.As(err, &apiErr) && apiErr.IsOrphanedSession() {
			fmt.Fprintf(os.Stderr, "Architect session is orphaned (tmux window was closed but session record remains).\n\n")
			fmt.Fprintf(os.Stderr, "Options:\n")
			fmt.Fprintf(os.Stderr, "  cortex architect --mode fresh   # Start a new session\n")
			fmt.Fprintf(os.Stderr, "  cortex architect --mode resume  # Resume the previous session\n")
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Attach to tmux session unless --detach is set
	if !architectDetachFlag {
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
		fmt.Printf("Architect running in tmux session: %s\n", resp.TmuxSession)
	}
}
