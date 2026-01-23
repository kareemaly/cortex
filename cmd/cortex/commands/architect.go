package commands

import (
	"fmt"
	"os"

	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/tmux"
	"github.com/spf13/cobra"
)

var (
	architectDetachFlag bool
	architectResumeFlag bool
	architectFreshFlag  bool
)

var architectCmd = &cobra.Command{
	Use:   "architect",
	Short: "Start or attach to architect session",
	Long: `Starts or attaches to the architect session for this project.

The architect lives in tmux window 0 and has full MCP tools for ticket
management and spawning ticket sessions.

Examples:
  cortex architect           # Start or attach to architect
  cortex architect --detach  # Start architect without attaching
  cortex architect --resume  # Resume an orphaned architect session
  cortex architect --fresh   # Clear existing session and start fresh`,
	Run: runArchitect,
}

func init() {
	architectCmd.Flags().BoolVar(&architectDetachFlag, "detach", false,
		"Spawn architect without attaching to session")
	architectCmd.Flags().BoolVar(&architectResumeFlag, "resume", false,
		"Resume an orphaned architect session")
	architectCmd.Flags().BoolVar(&architectFreshFlag, "fresh", false,
		"Clear existing session and start fresh")
	rootCmd.AddCommand(architectCmd)
}

func runArchitect(cmd *cobra.Command, args []string) {
	// Validate mutually exclusive flags
	if architectResumeFlag && architectFreshFlag {
		fmt.Fprintf(os.Stderr, "Error: --resume and --fresh are mutually exclusive\n")
		os.Exit(1)
	}

	// Get project path
	projectPath, err := resolveProjectPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Create SDK client
	client := sdk.DefaultClient(projectPath)

	// Determine mode from flags
	mode := ""
	if architectResumeFlag {
		mode = "resume"
	} else if architectFreshFlag {
		mode = "fresh"
	}

	// Spawn architect via SDK
	resp, err := client.SpawnArchitect(mode)
	if err != nil {
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
