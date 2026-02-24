package commands

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	architectconfig "github.com/kareemaly/cortex/internal/architect/config"
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

		projectName := filepath.Base(architectPath)
		if cfg, err := architectconfig.Load(architectPath); err == nil && cfg.Name != "" {
			projectName = cfg.Name
		}

		client := sdk.DefaultClient(architectPath)
		logBuf := tuilog.NewBuffer(tuilog.DefaultCapacity)
		p := tea.NewProgram(
			views.New(client, logBuf, projectName),
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
