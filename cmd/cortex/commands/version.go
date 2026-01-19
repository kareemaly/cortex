package commands

import (
	"fmt"

	"github.com/kareemaly/cortex1/pkg/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		info := version.Get()
		fmt.Printf("cortex %s\n", info.Version)
		fmt.Printf("  Commit:     %s\n", info.Commit)
		fmt.Printf("  Built:      %s\n", info.BuildDate)
		fmt.Printf("  Go version: %s\n", info.GoVersion)
		fmt.Printf("  Platform:   %s\n", info.Platform)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
