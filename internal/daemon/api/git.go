package api

import "os/exec"

// validateCommitSHAs returns the subset of shas that don't resolve in repoDir.
func validateCommitSHAs(repoDir string, shas []string) []string {
	var invalid []string
	for _, sha := range shas {
		cmd := exec.Command("git", "cat-file", "-e", sha)
		cmd.Dir = repoDir
		if err := cmd.Run(); err != nil {
			invalid = append(invalid, sha)
		}
	}
	return invalid
}
