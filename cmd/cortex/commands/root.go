package commands

import (
	"os"

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
