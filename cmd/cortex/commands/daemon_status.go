package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/kareemaly/cortex/internal/daemon/autostart"
	"github.com/spf13/cobra"
)

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon status",
	Long:  `Show the current status of the Cortex daemon.`,
	Run: func(cmd *cobra.Command, args []string) {
		status, err := autostart.GetStatus()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if !status.Running {
			fmt.Println("Daemon is not running")
			return
		}

		fmt.Println("Daemon is running")
		fmt.Printf("  PID:     %d\n", status.PID)
		fmt.Printf("  Port:    %d\n", status.Port)
		if status.Version != "" {
			fmt.Printf("  Version: %s\n", status.Version)
		}
		fmt.Printf("  Uptime:  %s\n", formatUptime(status.Uptime))
	},
}

func init() {
	daemonCmd.AddCommand(daemonStatusCmd)
}

// formatUptime formats a duration in a human-readable format.
func formatUptime(d time.Duration) string {
	d = d.Round(time.Second)

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh%dm%ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
