package commands

import (
	"fmt"

	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/spf13/cobra"
)

var architectDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Remove an architect from the registry",
	Long: `Remove an architect from the global registry.

This does not delete any files — it only removes the architect from tracking.
Use the architect's name or run from within the architect directory.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := ""
		if len(args) > 0 {
			name = args[0]
		}

		ensureDaemon()

		architectPath, err := resolveArchitectPath(name)
		if err != nil {
			return err
		}

		client := sdk.DefaultClient("")
		if err := client.UnlinkArchitect(architectPath); err != nil {
			return fmt.Errorf("failed to remove architect: %w", err)
		}

		fmt.Printf("Removed architect: %s\n", architectPath)
		return nil
	},
}

func init() {
	architectCmd.AddCommand(architectDeleteCmd)
}
