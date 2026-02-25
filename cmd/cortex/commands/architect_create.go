package commands

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/kareemaly/cortex/internal/cli/tui/wizard"
)

var (
	architectCreateGlobalOnly bool
	architectCreateForce      bool
	architectCreateAgent      string
)

var architectCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Initialize a new architect workspace",
	Long: `Initialize Cortex for an architect workspace.

Creates the global ~/.cortex/settings.yaml and sets up an architect workspace
in the current directory under a slugified subdirectory named after [name].

The agent is auto-detected from your PATH. If both claude and opencode are
available, you'll be prompted to choose. Use --agent to skip detection.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runArchitectCreate,
}

func init() {
	architectCreateCmd.Flags().BoolVarP(&architectCreateGlobalOnly, "global-only", "g", false, "Only set up global ~/.cortex/, skip project setup")
	architectCreateCmd.Flags().BoolVarP(&architectCreateForce, "force", "f", false, "Overwrite existing config files")
	architectCreateCmd.Flags().StringVarP(&architectCreateAgent, "agent", "a", "", "Agent type: claude, opencode (auto-detected if not set)")
	architectCmd.AddCommand(architectCreateCmd)
}

func runArchitectCreate(cmd *cobra.Command, args []string) error {
	isTTY := isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())

	if !isTTY && !architectCreateGlobalOnly && len(args) == 0 {
		return fmt.Errorf("name argument required in non-interactive mode")
	}

	var argName string
	if len(args) > 0 {
		argName = args[0]
	}

	flagAgent := ""
	if cmd.Flags().Changed("agent") {
		flagAgent = architectCreateAgent
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	cfg := wizard.Config{
		ArgName:    argName,
		FlagAgent:  flagAgent,
		FlagForce:  architectCreateForce,
		GlobalOnly: architectCreateGlobalOnly,
		IsTTY:      isTTY,
		Cwd:        cwd,
	}

	m := wizard.New(cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())

	final, err := p.Run()
	if err != nil {
		return err
	}

	result := final.(wizard.Model)
	return result.Err()
}
