package commands

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/kareemaly/cortex/internal/upgrade"
)

var (
	upgradeCheck   bool
	upgradeVersion string
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade cortex to the latest version",
	Long: `Upgrade cortex and cortexd binaries to the latest release from GitHub.

Examples:
  cortex upgrade           # Upgrade to latest version
  cortex upgrade --check   # Check if an update is available
  cortex upgrade -v 1.0.0  # Upgrade to a specific version`,
	RunE: runUpgrade,
}

func init() {
	upgradeCmd.Flags().BoolVarP(&upgradeCheck, "check", "c", false, "Only check if an update is available")
	upgradeCmd.Flags().StringVarP(&upgradeVersion, "version", "v", "", "Upgrade to a specific version")
	rootCmd.AddCommand(upgradeCmd)
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	opts := upgrade.Options{
		CheckOnly: upgradeCheck,
		Version:   upgradeVersion,
	}

	// Print check header
	fmt.Println("Checking for updates...")

	// Create callback for progress reporting
	callback := func(step string, success bool, message string) {
		switch step {
		case "checksums":
			// Don't print anything for checksums - too noisy
		case "download_cortex":
			if !success {
				fmt.Printf("\nDownloading %s...\n", upgrade.GetBinaryName("cortex"))
			} else {
				fmt.Printf("  %s Downloaded\n", checkMark())
			}
		case "verify_cortex":
			if success {
				fmt.Printf("  %s Verified checksum\n", checkMark())
			}
		case "download_cortexd":
			if !success {
				fmt.Printf("\nDownloading %s...\n", upgrade.GetBinaryName("cortexd"))
			} else {
				fmt.Printf("  %s Downloaded\n", checkMark())
			}
		case "verify_cortexd":
			if success {
				fmt.Printf("  %s Verified checksum\n", checkMark())
			}
		case "stop_daemon":
			if !success {
				fmt.Println("\nUpgrading:")
			}
			fmt.Printf("  %s %s\n", statusMark(success), message)
		case "backup_cortex", "backup_cortexd":
			fmt.Printf("  %s %s\n", statusMark(success), message)
		case "install_cortex", "install_cortexd":
			fmt.Printf("  %s %s\n", statusMark(success), message)
		case "codesign":
			if runtime.GOOS == "darwin" && success {
				fmt.Printf("  %s %s\n", checkMark(), message)
			}
		case "start_daemon":
			fmt.Printf("  %s %s\n", statusMark(success), message)
		}
	}

	result, err := upgrade.Run(opts, callback)
	if err != nil {
		return err
	}

	// Print version info
	fmt.Printf("  Current: %s\n", result.CurrentVersion)
	fmt.Printf("  Latest:  %s\n", result.LatestVersion)

	// Print final status
	if opts.CheckOnly {
		fmt.Println()
		if result.CurrentVersion == result.LatestVersion || result.CurrentVersion == "dev" && result.LatestVersion != "" {
			if result.CurrentVersion == "dev" {
				fmt.Printf("Running dev build. Latest release is %s.\n", result.LatestVersion)
				fmt.Println("Run 'cortex upgrade' to install the latest release.")
			} else {
				fmt.Println("Already up to date.")
			}
		} else {
			fmt.Println("Update available! Run 'cortex upgrade' to install.")
		}
		return nil
	}

	if result.AlreadyLatest {
		fmt.Println("\nAlready up to date.")
		return nil
	}

	if result.WasUpgraded {
		fmt.Printf("\nUpgrade complete! %s → %s\n", result.CurrentVersion, result.LatestVersion)
	}

	return nil
}

func statusMark(success bool) string {
	if success {
		return checkMark()
	}
	return crossMark()
}
