package commands

import (
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage the Cortex daemon",
	Long: `Manage the Cortex daemon process.

The daemon is automatically started when needed by other commands.
These subcommands let you manually control the daemon.

Examples:
  cortex daemon status   # Check if daemon is running
  cortex daemon stop     # Stop the daemon
  cortex daemon restart  # Restart the daemon
  cortex daemon logs     # View daemon logs
  cortex daemon logs -f  # Follow daemon logs`,
}

func init() {
	rootCmd.AddCommand(daemonCmd)
}
