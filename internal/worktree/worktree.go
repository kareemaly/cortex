package worktree

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// WorktreeBaseDir is the directory where worktrees are created.
const WorktreeBaseDir = ".cortex/worktrees"

// Manager handles git worktree operations.
type Manager struct {
	projectPath string
}

// NewManager creates a new worktree manager for the given project.
func NewManager(projectPath string) *Manager {
	return &Manager{projectPath: projectPath}
}

// Create creates a worktree with a feature branch.
// Returns the worktree path and branch name.
func (m *Manager) Create(ctx context.Context, sessionID, slug string) (worktreePath, branchName string, err error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("get home directory: %w", err)
	}

	worktreePath = filepath.Join(homeDir, WorktreeBaseDir, sessionID)
	branchName = "ticket/" + slug

	// Ensure base directory exists
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0755); err != nil {
		return "", "", fmt.Errorf("create worktree base dir: %w", err)
	}

	// Create worktree with new branch
	cmd := exec.CommandContext(ctx, "git", "worktree", "add", worktreePath, "-b", branchName)
	cmd.Dir = m.projectPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", "", fmt.Errorf("git worktree add: %s: %w", string(output), err)
	}

	return worktreePath, branchName, nil
}

// Remove removes a worktree and its branch.
func (m *Manager) Remove(ctx context.Context, worktreePath, branchName string) error {
	// Remove worktree
	cmd := exec.CommandContext(ctx, "git", "worktree", "remove", worktreePath, "--force")
	cmd.Dir = m.projectPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree remove: %s: %w", string(output), err)
	}

	// Delete branch
	cmd = exec.CommandContext(ctx, "git", "branch", "-D", branchName)
	cmd.Dir = m.projectPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git branch -D: %s: %w", string(output), err)
	}

	return nil
}
