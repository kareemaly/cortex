package commands

import (
	"fmt"
	"os"

	"github.com/kareemaly/cortex/internal/daemon/autostart"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cortex",
	Short: "Cortex - AI-powered development workflow",
	Long: `Cortex is an AI-powered development workflow tool that helps you
manage tickets, sessions, and project architecture with intelligent assistance.`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// ensureDaemon ensures the daemon is running before executing a command.
// If the daemon cannot be started, it prints an error and exits.
func ensureDaemon() {
	if err := autostart.EnsureDaemonRunning(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to start daemon: %v\n", err)
		os.Exit(1)
	}
}
