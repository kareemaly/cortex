package api

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// validateCommitSHAs returns the subset of shas that don't resolve in repoDir.
func validateCommitSHAs(repoDir string, shas []string) []string {
	dir := expandHome(repoDir)
	var invalid []string
	for _, sha := range shas {
		cmd := exec.Command("git", "cat-file", "-e", sha)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			invalid = append(invalid, sha)
		}
	}
	return invalid
}

// expandHome replaces a leading ~ with the user's home directory.
func expandHome(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[1:])
}
