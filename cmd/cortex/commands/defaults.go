package commands

import (
	"github.com/spf13/cobra"
)

var defaultsCmd = &cobra.Command{
	Use:   "defaults",
	Short: "Manage default configurations",
	Long:  `Commands for managing the default configuration files in ~/.cortex/defaults/.`,
}

func init() {
	defaultsCmd.AddCommand(defaultsUpgradeCmd)
	rootCmd.AddCommand(defaultsCmd)
}
