package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/kareemaly/cortex/internal/install"
)

func printItems(items []install.SetupItem) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "" // Acceptable fallback for display purposes only
	}

	for _, item := range items {
		path := item.Path
		// Replace home directory with ~
		if homeDir != "" && strings.HasPrefix(path, homeDir) {
			path = "~" + path[len(homeDir):]
		}

		switch item.Status {
		case install.StatusCreated:
			fmt.Printf("  %s Created %s\n", checkMark(), path)
		case install.StatusExists:
			fmt.Printf("  %s %s already exists\n", bullet(), path)
		case install.StatusSkipped:
			fmt.Printf("  - Skipped %s\n", path)
		}

		if item.Error != nil {
			fmt.Printf("    Error: %v\n", item.Error)
		}
	}
}

func checkMark() string {
	return "\u2713"
}

func crossMark() string {
	return "\u2717"
}

func bullet() string {
	return "\u2022"
}
