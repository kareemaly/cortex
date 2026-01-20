package commands

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cortexd",
	Short: "Cortex daemon - background services for cortex",
	Long: `Cortexd provides background services for the Cortex development workflow.

By default, it starts the HTTP API server. Use subcommands for other modes:
  cortexd serve  - Start the HTTP API server (default)
  cortexd mcp    - Start the MCP server for AI agent integration`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
