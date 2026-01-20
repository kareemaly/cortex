package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Repo represents a git repository.
type Repo struct {
	Root   string // Absolute path to repo root
	GitDir string // Path to .git directory
}

// IsGitInstalled checks if the git command is available.
func IsGitInstalled() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// FindRepo discovers the git repository containing the given path.
// Returns the repo root and git directory.
func FindRepo(path string) (*Repo, error) {
	if !IsGitInstalled() {
		return nil, &GitNotInstalledError{}
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	root, err := runGit(absPath, "rev-parse", "--show-toplevel")
	if err != nil {
		if isNotARepoError(err) {
			return nil, &NotARepoError{Path: absPath}
		}
		return nil, err
	}

	gitDir, err := runGit(absPath, "rev-parse", "--git-dir")
	if err != nil {
		return nil, err
	}

	// Make gitDir absolute if relative
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(root, gitDir)
	}

	return &Repo{
		Root:   root,
		GitDir: gitDir,
	}, nil
}

// DiscoverRepos finds git repositories at the given paths.
// Paths that are not git repos are silently skipped.
// Returns unique repos (deduped by root path).
func DiscoverRepos(paths []string) ([]*Repo, error) {
	if !IsGitInstalled() {
		return nil, &GitNotInstalledError{}
	}

	seen := make(map[string]bool)
	var repos []*Repo

	for _, path := range paths {
		repo, err := FindRepo(path)
		if err != nil {
			if IsNotARepo(err) {
				continue
			}
			return nil, err
		}
		if !seen[repo.Root] {
			seen[repo.Root] = true
			repos = append(repos, repo)
		}
	}

	return repos, nil
}

// ValidateRepo confirms the path contains a valid git repository.
func ValidateRepo(path string) error {
	if !IsGitInstalled() {
		return &GitNotInstalledError{}
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	_, err = runGit(absPath, "rev-parse", "--git-dir")
	if err != nil {
		if isNotARepoError(err) {
			return &NotARepoError{Path: absPath}
		}
		return err
	}

	return nil
}

// runGit executes a git command in the given directory and returns trimmed stdout.
func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", &gitError{
			args:   args,
			stderr: strings.TrimSpace(stderr.String()),
			err:    err,
		}
	}

	return strings.TrimSpace(stdout.String()), nil
}

// runGitLines executes a git command and returns output split by newlines.
func runGitLines(dir string, args ...string) ([]string, error) {
	output, err := runGit(dir, args...)
	if err != nil {
		return nil, err
	}

	if output == "" {
		return nil, nil
	}

	return strings.Split(output, "\n"), nil
}

// gitError wraps git command execution errors.
type gitError struct {
	args   []string
	stderr string
	err    error
}

func (e *gitError) Error() string {
	if e.stderr != "" {
		return fmt.Sprintf("git %s: %s", strings.Join(e.args, " "), e.stderr)
	}
	return fmt.Sprintf("git %s: %v", strings.Join(e.args, " "), e.err)
}

func (e *gitError) Unwrap() error {
	return e.err
}

// isNotARepoError checks if the error indicates we're not in a git repo.
func isNotARepoError(err error) bool {
	if gitErr, ok := err.(*gitError); ok {
		return strings.Contains(gitErr.stderr, "not a git repository")
	}
	return false
}

// isUnknownRevisionError checks if the error indicates an unknown revision.
func isUnknownRevisionError(err error) bool {
	if gitErr, ok := err.(*gitError); ok {
		return strings.Contains(gitErr.stderr, "unknown revision") ||
			strings.Contains(gitErr.stderr, "bad revision")
	}
	return false
}
