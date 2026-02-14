package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	projectconfig "github.com/kareemaly/cortex/internal/project/config"
	"github.com/kareemaly/cortex/internal/prompt"
)

var ejectForce bool

var ejectCmd = &cobra.Command{
	Use:   "eject <prompt-path>",
	Short: "Copy an inherited prompt to your project for customization",
	Long: `Copy a prompt file from the base configuration to .cortex/prompts/
for local customization. The prompt path is relative to the prompts directory.

Examples:
  cortex eject ticket/work/SYSTEM.md
  cortex eject architect/KICKOFF.md
  cortex eject ticket/work/SYSTEM.md --force`,
	Args: cobra.ExactArgs(1),
	RunE: runEject,
}

func init() {
	ejectCmd.Flags().BoolVarP(&ejectForce, "force", "f", false, "Overwrite existing file")
	rootCmd.AddCommand(ejectCmd)
}

func runEject(cmd *cobra.Command, args []string) error {
	// Parse and clean the prompt path
	promptPath := strings.TrimPrefix(args[0], "/")
	promptPath = filepath.Clean(promptPath)

	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Load project config
	cfg, projectRoot, err := projectconfig.LoadFromPath(cwd)
	if err != nil {
		if projectconfig.IsProjectNotFound(err) {
			return fmt.Errorf("not in a cortex project (no .cortex directory found)")
		}
		return err
	}

	// Get extend path
	extendPath := cfg.ResolvedExtendPath()
	if extendPath == "" {
		return fmt.Errorf("project has no 'extend' configured in .cortex/cortex.yaml")
	}

	// Resolve paths
	sourcePath := filepath.Join(prompt.BasePromptsDir(extendPath), promptPath)
	destPath := filepath.Join(prompt.PromptsDir(projectRoot), promptPath)

	// Validate source exists and is not a directory
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("source prompt not found: %s", formatPath(sourcePath))
		}
		return fmt.Errorf("failed to access source: %w", err)
	}
	if sourceInfo.IsDir() {
		return fmt.Errorf("cannot eject directory: %s", formatPath(sourcePath))
	}

	// Check if destination exists
	if _, err := os.Stat(destPath); err == nil {
		if !ejectForce {
			return fmt.Errorf("destination already exists: %s\n\nUse --force to overwrite", formatPath(destPath))
		}
	}

	// Create destination directory
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Copy the file
	if err := copyFile(sourcePath, destPath); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Print success
	fmt.Printf("Ejected: %s\n", promptPath)
	fmt.Printf("  From: %s\n", formatPath(sourcePath))
	fmt.Printf("  To:   %s\n", formatPath(destPath))

	return nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) (err error) {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := sourceFile.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := destFile.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// formatPath replaces home directory with ~ for display.
func formatPath(path string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, homeDir) {
		return "~" + path[len(homeDir):]
	}
	return path
}
