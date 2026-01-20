package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// InitTestRepo creates a temporary git repository for testing.
// Returns the repo path and a cleanup function.
func InitTestRepo(t *testing.T) (string, func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	cleanup := func() {
		_ = os.RemoveAll(dir)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		cleanup()
		t.Fatalf("git init: %v", err)
	}

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		cleanup()
		t.Fatalf("git config email: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		cleanup()
		t.Fatalf("git config name: %v", err)
	}

	return dir, cleanup
}

// CommitFile creates a file and commits it, returning the commit SHA.
func CommitFile(t *testing.T, repoPath, filename, content, message string) string {
	t.Helper()

	// Create file
	filePath := filepath.Join(repoPath, filename)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	// Stage file
	cmd := exec.Command("git", "add", filename)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("git add: %v", err)
	}

	// Commit
	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	// Get commit SHA
	sha, err := runGit(repoPath, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("get commit sha: %v", err)
	}

	return sha
}

// CreateBranch creates a new branch.
func CreateBranch(t *testing.T, repoPath, branchName string) {
	t.Helper()

	cmd := exec.Command("git", "branch", branchName)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("git branch: %v", err)
	}
}

// CheckoutBranch switches to a branch.
func CheckoutBranch(t *testing.T, repoPath, branchName string) {
	t.Helper()

	cmd := exec.Command("git", "checkout", branchName)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("git checkout: %v", err)
	}
}
