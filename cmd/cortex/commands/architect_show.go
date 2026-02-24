package commands

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/cli/tui/tuilog"
	"github.com/kareemaly/cortex/internal/cli/tui/views"
	"github.com/spf13/cobra"
)

var architectShowCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Open architect TUI",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := ""
		if len(args) > 0 {
			name = args[0]
		}

		ensureDaemon()

		architectPath, err := resolveArchitectPath(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		client := sdk.DefaultClient(architectPath)
		logBuf := tuilog.NewBuffer(tuilog.DefaultCapacity)
		p := tea.NewProgram(
			views.New(client, logBuf),
			tea.WithAltScreen(),
		)
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	architectCmd.AddCommand(architectShowCmd)
}
