package commands

import (
	"fmt"
	"os"

	"github.com/kareemaly/cortex/internal/daemon/autostart"
	"github.com/spf13/cobra"
)

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the daemon",
	Long:  `Stop the running Cortex daemon.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := autostart.StopDaemon(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Daemon stopped")
	},
}

func init() {
	daemonCmd.AddCommand(daemonStopCmd)
}
