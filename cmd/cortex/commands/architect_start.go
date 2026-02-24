package commands

import (
	"errors"
	"fmt"

	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/tmux"
	"github.com/spf13/cobra"
)

var (
	architectStartMode   string
	architectStartDetach bool
)

var architectStartCmd = &cobra.Command{
	Use:   "start [name]",
	Short: "Start or attach to an architect session",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runArchitectStart,
}

func init() {
	architectStartCmd.Flags().StringVar(&architectStartMode, "mode", "", "Session mode: normal, fresh, or resume")
	architectStartCmd.Flags().BoolVar(&architectStartDetach, "detach", false, "Start in detached mode")
	architectCmd.AddCommand(architectStartCmd)
}

func runArchitectStart(cmd *cobra.Command, args []string) error {
	name := ""
	if len(args) > 0 {
		name = args[0]
	}

	ensureDaemon()

	architectPath, err := resolveArchitectPath(name)
	if err != nil {
		return err
	}

	client := sdk.DefaultClient(architectPath)

	// Check daemon health
	if err := client.Health(); err != nil {
		return fmt.Errorf("daemon not running: %w\nStart it with: cortexd", err)
	}

	// Try to spawn architect
	resp, err := client.SpawnArchitect(architectStartMode)
	if err != nil {
		apiErr, ok := err.(*sdk.APIError)
		if ok && apiErr.IsOrphanedSession() {
			return fmt.Errorf("architect session is orphaned. Use --mode resume to continue or --mode fresh to start over\n  cortex architect start --mode fresh")
		}
		return fmt.Errorf("failed to spawn architect: %w", err)
	}

	if architectStartDetach {
		fmt.Printf("Architect started in tmux session '%s', window '%s'\n", resp.TmuxSession, resp.TmuxWindow)
		return nil
	}

	// Attach to tmux session
	manager, err := tmux.NewManager()
	if err != nil {
		if tmux.IsNotInstalled(err) {
			return fmt.Errorf("tmux is required but not installed")
		}
		return fmt.Errorf("failed to create tmux manager: %w", err)
	}

	if err := manager.AttachSession(resp.TmuxSession); err != nil {
		var _ = errors.New // ensure errors is used
		return fmt.Errorf("failed to attach to session: %w", err)
	}
	return nil
}
