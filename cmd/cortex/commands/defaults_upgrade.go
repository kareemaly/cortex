package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/kareemaly/cortex/internal/install"
)

var (
	defaultsUpgradeDryRun bool
	defaultsUpgradeYes    bool
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
	defaultsUpgradeCmd.Flags().BoolVarP(&defaultsUpgradeYes, "yes", "y", false, "Skip confirmation prompt")
}

// defaultConfigs lists all config directories to upgrade.
var defaultConfigs = []string{"main"}

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
	// First, collect all changes across all configs
	type configChanges struct {
		name      string
		targetDir string
		items     []install.CompareItem
	}
	var allChanges []configChanges
	var totalUpdates, totalCreates, totalUnchanged int

	fmt.Println("Checking for updates...")
	fmt.Println()

	for _, configName := range defaultConfigs {
		targetDir := fmt.Sprintf("%s/.cortex/defaults/%s", homeDir, configName)
		items, err := install.CompareEmbeddedDefaults(configName, targetDir)
		if err != nil {
			return fmt.Errorf("failed to compare defaults for %s: %w", configName, err)
		}
		allChanges = append(allChanges, configChanges{
			name:      configName,
			targetDir: targetDir,
			items:     items,
		})

		for _, item := range items {
			if item.IsDir {
				continue
			}
			switch item.Status {
			case install.CompareWillUpdate:
				totalUpdates++
			case install.CompareWillCreate:
				totalCreates++
			case install.CompareUnchanged:
				totalUnchanged++
			}
		}
	}

	// If no changes, exit early
	if totalUpdates == 0 && totalCreates == 0 {
		fmt.Println("All defaults are already up to date.")
		return nil
	}

	// Display changes for each config
	for _, cc := range allChanges {
		fmt.Printf("=== %s ===\n\n", cc.name)
		fmt.Println("Files:")

		for _, item := range cc.items {
			if item.IsDir {
				continue
			}

			path := defaultsFormatPath(item.Path)

			switch item.Status {
			case install.CompareWillUpdate:
				fmt.Printf("  %s %s (will update)\n", bullet(), path)
			case install.CompareWillCreate:
				fmt.Printf("  + %s (will create)\n", path)
			case install.CompareUnchanged:
				fmt.Printf("  %s %s (unchanged)\n", checkMark(), path)
			}

			if item.Error != nil {
				fmt.Printf("    Error: %v\n", item.Error)
			}
		}
		fmt.Println()
	}

	// Show diffs for files that will be updated
	hasDiffs := false
	for _, cc := range allChanges {
		for _, item := range cc.items {
			if item.Status == install.CompareWillUpdate && !item.IsDir {
				if !hasDiffs {
					fmt.Println("Changes:")
					fmt.Println()
					hasDiffs = true
				}
				diff := generateUnifiedDiff(item.DiskContent, item.EmbeddedContent, defaultsFormatPath(item.Path))
				fmt.Println(diff)
			}
		}
	}

	fmt.Printf("Summary: %d to update, %d to create, %d unchanged\n", totalUpdates, totalCreates, totalUnchanged)
	fmt.Println()

	// Prompt for confirmation unless --yes flag is set
	if !defaultsUpgradeYes {
		if !isatty.IsTerminal(os.Stdin.Fd()) && !isatty.IsCygwinTerminal(os.Stdin.Fd()) {
			return fmt.Errorf("stdin is not a terminal; use --yes to skip confirmation")
		}

		if !promptConfirmation("Proceed with upgrade? [y/N]: ") {
			fmt.Println("Upgrade cancelled.")
			return nil
		}
	}

	// Apply changes
	fmt.Println()
	fmt.Println("Applying changes...")

	for _, cc := range allChanges {
		if _, err := install.CopyEmbeddedDefaults(cc.name, cc.targetDir, true); err != nil {
			return fmt.Errorf("failed to upgrade defaults for %s: %w", cc.name, err)
		}
	}

	fmt.Printf("Upgraded %d files.\n", totalUpdates+totalCreates)

	// Clean up legacy directories if they exist
	legacyDirs := []string{
		filepath.Join(homeDir, ".cortex", "defaults", "claude-code"),
		filepath.Join(homeDir, ".cortex", "defaults", "opencode"),
	}
	for _, dir := range legacyDirs {
		if _, err := os.Stat(dir); err == nil {
			if err := os.RemoveAll(dir); err == nil {
				fmt.Printf("Removed legacy directory: %s\n", defaultsFormatPath(dir))
			}
		}
	}

	fmt.Println()
	fmt.Println("Note: If you have existing projects with 'extend: ~/.cortex/defaults/claude-code'")
	fmt.Println("or 'extend: ~/.cortex/defaults/opencode', update them to 'extend: ~/.cortex/defaults/main'.")

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

// promptConfirmation prompts the user for y/n confirmation.
func promptConfirmation(message string) bool {
	fmt.Print(message)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}

// generateUnifiedDiff generates a unified diff between old and new content.
func generateUnifiedDiff(oldContent, newContent []byte, path string) string {
	oldLines := strings.Split(string(oldContent), "\n")
	newLines := strings.Split(string(newContent), "\n")

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("--- %s\n", path))
	sb.WriteString(fmt.Sprintf("+++ %s (embedded)\n", path))

	// Simple diff: find contiguous regions of changes
	// This is a basic implementation that shows context around changes
	const contextLines = 3

	type hunk struct {
		oldStart, oldCount int
		newStart, newCount int
		lines              []string
	}

	var hunks []hunk
	i, j := 0, 0

	for i < len(oldLines) || j < len(newLines) {
		// Find next difference
		for i < len(oldLines) && j < len(newLines) && oldLines[i] == newLines[j] {
			i++
			j++
		}

		if i >= len(oldLines) && j >= len(newLines) {
			break
		}

		// Found a difference, build a hunk
		hunkOldStart := max(0, i-contextLines)
		hunkNewStart := max(0, j-contextLines)

		var hunkLines []string

		// Add leading context
		for k := hunkOldStart; k < i; k++ {
			hunkLines = append(hunkLines, " "+oldLines[k])
		}

		// Find the extent of the difference
		diffStartI, diffStartJ := i, j

		// Consume differing lines until we find contextLines of matching lines
		for {
			// Skip differing lines in old
			for i < len(oldLines) && (j >= len(newLines) || oldLines[i] != newLines[j]) {
				hunkLines = append(hunkLines, "-"+oldLines[i])
				i++
			}
			// Skip differing lines in new
			for j < len(newLines) && (i >= len(oldLines) || oldLines[i] != newLines[j]) {
				hunkLines = append(hunkLines, "+"+newLines[j])
				j++
			}

			// Check if we have enough matching context to end this hunk
			matchCount := 0
			for i+matchCount < len(oldLines) && j+matchCount < len(newLines) &&
				oldLines[i+matchCount] == newLines[j+matchCount] {
				matchCount++
				if matchCount >= contextLines*2 {
					break
				}
			}

			if matchCount >= contextLines*2 || (i >= len(oldLines) && j >= len(newLines)) {
				// End of hunk - add trailing context
				trailingContext := min(contextLines, matchCount)
				for k := 0; k < trailingContext; k++ {
					hunkLines = append(hunkLines, " "+oldLines[i+k])
				}
				i += trailingContext
				j += trailingContext
				break
			}

			// Not enough context, continue the hunk
			for k := 0; k < matchCount; k++ {
				hunkLines = append(hunkLines, " "+oldLines[i])
				i++
				j++
			}
		}

		// Calculate hunk sizes
		oldCount := i - hunkOldStart
		newCount := j - hunkNewStart

		// Only add non-empty hunks
		if len(hunkLines) > 0 && (diffStartI != i || diffStartJ != j) {
			hunks = append(hunks, hunk{
				oldStart: hunkOldStart + 1, // 1-indexed
				oldCount: oldCount,
				newStart: hunkNewStart + 1, // 1-indexed
				newCount: newCount,
				lines:    hunkLines,
			})
		}
	}

	// Format hunks
	for _, h := range hunks {
		sb.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", h.oldStart, h.oldCount, h.newStart, h.newCount))
		for _, line := range h.lines {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
