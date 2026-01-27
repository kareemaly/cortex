package commands

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/cli/tui/dashboard"
	"github.com/kareemaly/cortex/internal/cli/tui/tuilog"
	"github.com/spf13/cobra"
)

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Open the global dashboard TUI",
	Long: `Opens a TUI dashboard showing all registered projects with their
active sessions. Navigate projects, expand to see running agents,
focus tmux windows, and spawn architects.

This is a global command — no project context required.

Examples:
  cortex dashboard   # Open the dashboard`,
	Run: func(cmd *cobra.Command, args []string) {
		client := sdk.DefaultClient("")
		logBuf := tuilog.NewBuffer(tuilog.DefaultCapacity)
		p := tea.NewProgram(
			dashboard.New(client, logBuf),
			tea.WithAltScreen(),
		)
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(dashboardCmd)
}
