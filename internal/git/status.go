package git

// GetBranch returns the current branch name.
// Returns "HEAD" if in detached HEAD state.
func GetBranch(repoPath string) (string, error) {
	branch, err := runGit(repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		if isNotARepoError(err) {
			return "", &NotARepoError{Path: repoPath}
		}
		return "", err
	}
	return branch, nil
}

// GetCommitSHA returns the HEAD commit SHA.
// If short is true, returns the short form (typically 7 characters).
func GetCommitSHA(repoPath string, short bool) (string, error) {
	args := []string{"rev-parse"}
	if short {
		args = append(args, "--short")
	}
	args = append(args, "HEAD")

	sha, err := runGit(repoPath, args...)
	if err != nil {
		if isNotARepoError(err) {
			return "", &NotARepoError{Path: repoPath}
		}
		return "", err
	}
	return sha, nil
}

// IsClean returns true if the working directory has no uncommitted changes.
// This includes both staged and unstaged changes.
func IsClean(repoPath string) (bool, error) {
	output, err := runGit(repoPath, "status", "--porcelain")
	if err != nil {
		if isNotARepoError(err) {
			return false, &NotARepoError{Path: repoPath}
		}
		return false, err
	}
	return output == "", nil
}

// GetBranchList returns a list of all local branch names.
func GetBranchList(repoPath string) ([]string, error) {
	branches, err := runGitLines(repoPath, "branch", "--format=%(refname:short)")
	if err != nil {
		if isNotARepoError(err) {
			return nil, &NotARepoError{Path: repoPath}
		}
		return nil, err
	}
	return branches, nil
}
