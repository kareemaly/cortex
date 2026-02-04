package commands

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/cli/tui/kanban"
	"github.com/kareemaly/cortex/internal/cli/tui/tuilog"
	projectconfig "github.com/kareemaly/cortex/internal/project/config"
	"github.com/spf13/cobra"
)

var kanbanCmd = &cobra.Command{
	Use:   "kanban",
	Short: "Open kanban TUI",
	Run: func(cmd *cobra.Command, args []string) {
		ensureDaemon()

		projectPath, err := resolveProjectPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		client := sdk.DefaultClient(projectPath)
		logBuf := tuilog.NewBuffer(tuilog.DefaultCapacity)
		p := tea.NewProgram(
			kanban.New(client, logBuf),
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

// resolveProjectPath finds the project root from the current directory.
func resolveProjectPath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	_, projectRoot, err := projectconfig.LoadFromPath(cwd)
	if err != nil {
		if projectconfig.IsProjectNotFound(err) {
			return "", fmt.Errorf("not in a cortex project (no .cortex directory found)")
		}
		return "", err
	}
	return projectRoot, nil
}
