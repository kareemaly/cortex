package install

import (
	"os/exec"
	"path/filepath"
	"strings"
)

// DetectProjectName attempts to detect the project name.
// First tries to extract from git remote origin, falls back to directory name.
func DetectProjectName(path string) string {
	// Try git remote origin
	if name := detectFromGitRemote(path); name != "" {
		return name
	}

	// Fallback to directory name
	absPath, err := filepath.Abs(path)
	if err != nil {
		return filepath.Base(path)
	}
	return filepath.Base(absPath)
}

// detectFromGitRemote extracts repo name from git remote origin URL.
func detectFromGitRemote(path string) string {
	cmd := exec.Command("git", "-C", path, "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	url := strings.TrimSpace(string(output))
	return parseRepoName(url)
}

// parseRepoName extracts the repository name from a git URL.
// Handles both SSH (git@github.com:user/repo.git) and HTTPS (https://github.com/user/repo.git) URLs.
func parseRepoName(url string) string {
	if url == "" {
		return ""
	}

	// Remove trailing .git
	url = strings.TrimSuffix(url, ".git")

	// Handle SSH URLs (git@github.com:user/repo)
	if strings.Contains(url, ":") && strings.Contains(url, "@") {
		parts := strings.Split(url, ":")
		if len(parts) == 2 {
			pathParts := strings.Split(parts[1], "/")
			if len(pathParts) > 0 {
				return pathParts[len(pathParts)-1]
			}
		}
	}

	// Handle HTTPS URLs (https://github.com/user/repo)
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return ""
}
