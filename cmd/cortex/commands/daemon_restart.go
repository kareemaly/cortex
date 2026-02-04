package commands

import (
	"fmt"
	"os"

	"github.com/kareemaly/cortex/internal/daemon/autostart"
	"github.com/spf13/cobra"
)

var daemonRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the daemon",
	Long:  `Stop and restart the Cortex daemon.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Stop daemon (ignore "not running" errors)
		_ = autostart.StopDaemon()

		// Start daemon
		if err := autostart.EnsureDaemonRunning(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Daemon restarted")
	},
}

func init() {
	daemonCmd.AddCommand(daemonRestartCmd)
}
