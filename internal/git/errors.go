package git

import "fmt"

// GitNotInstalledError indicates the git binary was not found.
type GitNotInstalledError struct{}

func (e *GitNotInstalledError) Error() string {
	return "git is not installed or not in PATH"
}

// NotARepoError indicates the path is not inside a git repository.
type NotARepoError struct {
	Path string
}

func (e *NotARepoError) Error() string {
	return fmt.Sprintf("not a git repository: %s", e.Path)
}

// CommitNotFoundError indicates a commit SHA does not exist.
type CommitNotFoundError struct {
	SHA string
}

func (e *CommitNotFoundError) Error() string {
	return fmt.Sprintf("commit not found: %s", e.SHA)
}

// IsNotInstalled returns true if err is a GitNotInstalledError.
func IsNotInstalled(err error) bool {
	_, ok := err.(*GitNotInstalledError)
	return ok
}

// IsNotARepo returns true if err is a NotARepoError.
func IsNotARepo(err error) bool {
	_, ok := err.(*NotARepoError)
	return ok
}

// IsCommitNotFound returns true if err is a CommitNotFoundError.
func IsCommitNotFound(err error) bool {
	_, ok := err.(*CommitNotFoundError)
	return ok
}
