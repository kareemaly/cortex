package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kareemaly/cortex/internal/install"
)

var (
	defaultsUpgradeDryRun bool
)

var defaultsUpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade default configurations",
	Long: `Refresh ~/.cortex/defaults/ with the latest embedded defaults from the binary.

This command updates the default configuration files and prompts to match
the version bundled with your current cortex binary. Use --dry-run to
preview changes without applying them.`,
	RunE: runDefaultsUpgradeCmd,
}

func init() {
	defaultsUpgradeCmd.Flags().BoolVarP(&defaultsUpgradeDryRun, "dry-run", "n", false, "Preview changes without applying them")
}

// defaultConfigs lists all config directories to upgrade.
var defaultConfigs = []string{"claude-code", "copilot"}

func runDefaultsUpgradeCmd(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	if defaultsUpgradeDryRun {
		return runDefaultsUpgradeDryRunAll(homeDir)
	}

	return runDefaultsUpgradeApplyAll(homeDir)
}

func runDefaultsUpgradeDryRunAll(homeDir string) error {
	fmt.Println("Dry run - no changes will be made")
	fmt.Println()

	for _, configName := range defaultConfigs {
		targetDir := fmt.Sprintf("%s/.cortex/defaults/%s", homeDir, configName)
		fmt.Printf("=== %s ===\n\n", configName)
		if err := runDefaultsUpgradeDryRun(configName, targetDir); err != nil {
			return err
		}
		fmt.Println()
	}

	return nil
}

func runDefaultsUpgradeApplyAll(homeDir string) error {
	fmt.Println("Upgrading defaults...")
	fmt.Println()

	for _, configName := range defaultConfigs {
		targetDir := fmt.Sprintf("%s/.cortex/defaults/%s", homeDir, configName)
		fmt.Printf("=== %s ===\n\n", configName)
		if err := runDefaultsUpgradeApply(configName, targetDir); err != nil {
			return err
		}
		fmt.Println()
	}

	return nil
}

func runDefaultsUpgradeDryRun(configName, targetDir string) error {
	items, err := install.CompareEmbeddedDefaults(configName, targetDir)
	if err != nil {
		return fmt.Errorf("failed to compare defaults: %w", err)
	}

	var toUpdate, toCreate, unchanged int

	fmt.Println("Files:")
	for _, item := range items {
		if item.IsDir {
			continue // Skip directories in output
		}

		path := defaultsFormatPath(item.Path)

		switch item.Status {
		case install.CompareWillUpdate:
			fmt.Printf("  %s %s (will update)\n", bullet(), path)
			toUpdate++
		case install.CompareWillCreate:
			fmt.Printf("  + %s (will create)\n", path)
			toCreate++
		case install.CompareUnchanged:
			fmt.Printf("  %s %s (unchanged)\n", checkMark(), path)
			unchanged++
		}

		if item.Error != nil {
			fmt.Printf("    Error: %v\n", item.Error)
		}
	}

	fmt.Println()
	fmt.Printf("Summary: %d to update, %d to create, %d unchanged\n", toUpdate, toCreate, unchanged)

	return nil
}

func runDefaultsUpgradeApply(configName, targetDir string) error {
	items, err := install.CopyEmbeddedDefaults(configName, targetDir, true)
	if err != nil {
		return fmt.Errorf("failed to upgrade defaults: %w", err)
	}

	var fileCount int

	fmt.Println("Files:")
	for _, item := range items {
		path := defaultsFormatPath(item.Path)

		// Skip directories in output
		info, statErr := os.Stat(item.Path)
		if statErr == nil && info.IsDir() {
			continue
		}

		if item.Status == install.StatusCreated {
			fmt.Printf("  %s Updated %s\n", checkMark(), path)
			fileCount++
		}

		if item.Error != nil {
			fmt.Printf("    Error: %v\n", item.Error)
		}
	}

	fmt.Println()
	fmt.Printf("Upgraded %d files\n", fileCount)

	return nil
}

// defaultsFormatPath replaces home directory with ~ for cleaner output.
func defaultsFormatPath(path string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, homeDir) {
		return "~" + path[len(homeDir):]
	}
	return path
}
