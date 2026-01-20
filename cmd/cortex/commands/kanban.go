package commands

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kareemaly/cortex1/internal/cli/sdk"
	"github.com/kareemaly/cortex1/internal/cli/tui/kanban"
	"github.com/spf13/cobra"
)

var kanbanCmd = &cobra.Command{
	Use:   "kanban",
	Short: "Open kanban TUI",
	Run: func(cmd *cobra.Command, args []string) {
		client := sdk.DefaultClient()
		p := tea.NewProgram(
			kanban.New(client),
			tea.WithAltScreen(),
		)
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(kanbanCmd)
}
