package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsGitInstalled(t *testing.T) {
	// Git should be installed in the test environment
	if !IsGitInstalled() {
		t.Skip("git is not installed")
	}
}

func TestFindRepo(t *testing.T) {
	repoPath, cleanup := InitTestRepo(t)
	defer cleanup()

	// Resolve symlinks for comparison (macOS /var -> /private/var)
	repoPathResolved, _ := filepath.EvalSymlinks(repoPath)

	// Create initial commit so repo is valid
	CommitFile(t, repoPath, "README.md", "# Test", "Initial commit")

	t.Run("from root", func(t *testing.T) {
		repo, err := FindRepo(repoPath)
		if err != nil {
			t.Fatalf("FindRepo: %v", err)
		}
		// Resolve symlinks in result as well
		gotResolved, _ := filepath.EvalSymlinks(repo.Root)
		if gotResolved != repoPathResolved {
			t.Errorf("Root = %q, want %q", gotResolved, repoPathResolved)
		}
	})

	t.Run("from subdirectory", func(t *testing.T) {
		subdir := filepath.Join(repoPath, "subdir")
		if err := os.MkdirAll(subdir, 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		repo, err := FindRepo(subdir)
		if err != nil {
			t.Fatalf("FindRepo: %v", err)
		}
		gotResolved, _ := filepath.EvalSymlinks(repo.Root)
		if gotResolved != repoPathResolved {
			t.Errorf("Root = %q, want %q", gotResolved, repoPathResolved)
		}
	})

	t.Run("not a repo", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "not-a-repo-*")
		if err != nil {
			t.Fatalf("create temp dir: %v", err)
		}
		t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

		_, err = FindRepo(tmpDir)
		if !IsNotARepo(err) {
			t.Errorf("expected NotARepoError, got %v", err)
		}
	})
}

func TestDiscoverRepos(t *testing.T) {
	repo1, cleanup1 := InitTestRepo(t)
	defer cleanup1()
	CommitFile(t, repo1, "README.md", "# Repo 1", "Initial commit")

	repo2, cleanup2 := InitTestRepo(t)
	defer cleanup2()
	CommitFile(t, repo2, "README.md", "# Repo 2", "Initial commit")

	notARepo, err := os.MkdirTemp("", "not-a-repo-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(notARepo) })

	t.Run("multiple repos", func(t *testing.T) {
		repos, err := DiscoverRepos([]string{repo1, repo2})
		if err != nil {
			t.Fatalf("DiscoverRepos: %v", err)
		}
		if len(repos) != 2 {
			t.Errorf("got %d repos, want 2", len(repos))
		}
	})

	t.Run("skips non-repos", func(t *testing.T) {
		repos, err := DiscoverRepos([]string{repo1, notARepo})
		if err != nil {
			t.Fatalf("DiscoverRepos: %v", err)
		}
		if len(repos) != 1 {
			t.Errorf("got %d repos, want 1", len(repos))
		}
	})

	t.Run("deduplicates", func(t *testing.T) {
		subdir := filepath.Join(repo1, "subdir")
		if err := os.MkdirAll(subdir, 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		repos, err := DiscoverRepos([]string{repo1, subdir})
		if err != nil {
			t.Fatalf("DiscoverRepos: %v", err)
		}
		if len(repos) != 1 {
			t.Errorf("got %d repos, want 1 (should dedupe)", len(repos))
		}
	})
}

func TestValidateRepo(t *testing.T) {
	repoPath, cleanup := InitTestRepo(t)
	defer cleanup()
	CommitFile(t, repoPath, "README.md", "# Test", "Initial commit")

	t.Run("valid repo", func(t *testing.T) {
		if err := ValidateRepo(repoPath); err != nil {
			t.Errorf("ValidateRepo: %v", err)
		}
	})

	t.Run("not a repo", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "not-a-repo-*")
		if err != nil {
			t.Fatalf("create temp dir: %v", err)
		}
		t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

		err = ValidateRepo(tmpDir)
		if !IsNotARepo(err) {
			t.Errorf("expected NotARepoError, got %v", err)
		}
	})
}

func TestGetBranch(t *testing.T) {
	repoPath, cleanup := InitTestRepo(t)
	defer cleanup()
	CommitFile(t, repoPath, "README.md", "# Test", "Initial commit")

	t.Run("default branch", func(t *testing.T) {
		branch, err := GetBranch(repoPath)
		if err != nil {
			t.Fatalf("GetBranch: %v", err)
		}
		// Default branch could be "main" or "master" depending on git config
		if branch != "main" && branch != "master" {
			t.Errorf("Branch = %q, want 'main' or 'master'", branch)
		}
	})

	t.Run("feature branch", func(t *testing.T) {
		CreateBranch(t, repoPath, "feature/test")
		CheckoutBranch(t, repoPath, "feature/test")

		branch, err := GetBranch(repoPath)
		if err != nil {
			t.Fatalf("GetBranch: %v", err)
		}
		if branch != "feature/test" {
			t.Errorf("Branch = %q, want 'feature/test'", branch)
		}
	})
}

func TestGetCommitSHA(t *testing.T) {
	repoPath, cleanup := InitTestRepo(t)
	defer cleanup()
	sha := CommitFile(t, repoPath, "README.md", "# Test", "Initial commit")

	t.Run("full SHA", func(t *testing.T) {
		got, err := GetCommitSHA(repoPath, false)
		if err != nil {
			t.Fatalf("GetCommitSHA: %v", err)
		}
		if got != sha {
			t.Errorf("SHA = %q, want %q", got, sha)
		}
	})

	t.Run("short SHA", func(t *testing.T) {
		got, err := GetCommitSHA(repoPath, true)
		if err != nil {
			t.Fatalf("GetCommitSHA: %v", err)
		}
		if len(got) < 7 || len(got) > 12 {
			t.Errorf("short SHA length = %d, want 7-12", len(got))
		}
		if got != sha[:len(got)] {
			t.Errorf("short SHA = %q, want prefix of %q", got, sha)
		}
	})
}

func TestIsClean(t *testing.T) {
	repoPath, cleanup := InitTestRepo(t)
	defer cleanup()
	CommitFile(t, repoPath, "README.md", "# Test", "Initial commit")

	t.Run("clean repo", func(t *testing.T) {
		clean, err := IsClean(repoPath)
		if err != nil {
			t.Fatalf("IsClean: %v", err)
		}
		if !clean {
			t.Error("expected clean repo")
		}
	})

	t.Run("untracked file", func(t *testing.T) {
		if err := os.WriteFile(filepath.Join(repoPath, "untracked.txt"), []byte("test"), 0644); err != nil {
			t.Fatalf("write file: %v", err)
		}

		clean, err := IsClean(repoPath)
		if err != nil {
			t.Fatalf("IsClean: %v", err)
		}
		if clean {
			t.Error("expected dirty repo (untracked file)")
		}
	})
}

func TestGetBranchList(t *testing.T) {
	repoPath, cleanup := InitTestRepo(t)
	defer cleanup()
	CommitFile(t, repoPath, "README.md", "# Test", "Initial commit")

	CreateBranch(t, repoPath, "feature/one")
	CreateBranch(t, repoPath, "feature/two")

	branches, err := GetBranchList(repoPath)
	if err != nil {
		t.Fatalf("GetBranchList: %v", err)
	}

	if len(branches) < 3 {
		t.Errorf("got %d branches, want at least 3", len(branches))
	}

	// Check that our branches exist
	branchSet := make(map[string]bool)
	for _, b := range branches {
		branchSet[b] = true
	}

	if !branchSet["feature/one"] {
		t.Error("missing branch 'feature/one'")
	}
	if !branchSet["feature/two"] {
		t.Error("missing branch 'feature/two'")
	}
}

func TestGetDiffStats(t *testing.T) {
	repoPath, cleanup := InitTestRepo(t)
	defer cleanup()

	baseSHA := CommitFile(t, repoPath, "README.md", "# Test\n", "Initial commit")

	// Make changes
	CommitFile(t, repoPath, "README.md", "# Test\n\nUpdated content\n", "Update readme")
	CommitFile(t, repoPath, "new.txt", "New file\n", "Add new file")

	t.Run("stats from base", func(t *testing.T) {
		stats, err := GetDiffStats(repoPath, baseSHA)
		if err != nil {
			t.Fatalf("GetDiffStats: %v", err)
		}

		if stats.FilesChanged != 2 {
			t.Errorf("FilesChanged = %d, want 2", stats.FilesChanged)
		}
		if stats.Insertions < 2 {
			t.Errorf("Insertions = %d, want >= 2", stats.Insertions)
		}
	})

	t.Run("unknown commit", func(t *testing.T) {
		_, err := GetDiffStats(repoPath, "deadbeef")
		if !IsCommitNotFound(err) {
			t.Errorf("expected CommitNotFoundError, got %v", err)
		}
	})
}

func TestGetChangedFiles(t *testing.T) {
	repoPath, cleanup := InitTestRepo(t)
	defer cleanup()

	// Create initial commit with two files
	CommitFile(t, repoPath, "README.md", "# Test\n", "Initial commit")
	baseSHA := CommitFile(t, repoPath, "keep.txt", "Keep this\n", "Add keep file")

	// Make changes
	CommitFile(t, repoPath, "README.md", "# Updated\n", "Update readme")
	CommitFile(t, repoPath, "new.txt", "New file\n", "Add new file")

	// Delete a file
	if err := os.Remove(filepath.Join(repoPath, "keep.txt")); err != nil {
		t.Fatalf("remove file: %v", err)
	}
	if _, err := runGit(repoPath, "add", "-A"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if _, err := runGit(repoPath, "commit", "-m", "Delete keep file"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	t.Run("changed files", func(t *testing.T) {
		files, err := GetChangedFiles(repoPath, baseSHA)
		if err != nil {
			t.Fatalf("GetChangedFiles: %v", err)
		}

		fileMap := make(map[string]string)
		for _, f := range files {
			fileMap[f.Path] = f.Status
		}

		if fileMap["README.md"] != "M" {
			t.Errorf("README.md status = %q, want 'M'", fileMap["README.md"])
		}
		if fileMap["new.txt"] != "A" {
			t.Errorf("new.txt status = %q, want 'A'", fileMap["new.txt"])
		}
		if fileMap["keep.txt"] != "D" {
			t.Errorf("keep.txt status = %q, want 'D'", fileMap["keep.txt"])
		}
	})
}

func TestErrorTypes(t *testing.T) {
	t.Run("GitNotInstalledError", func(t *testing.T) {
		err := &GitNotInstalledError{}
		if err.Error() == "" {
			t.Error("expected non-empty error message")
		}
		if !IsNotInstalled(err) {
			t.Error("IsNotInstalled should return true")
		}
	})

	t.Run("NotARepoError", func(t *testing.T) {
		err := &NotARepoError{Path: "/some/path"}
		if err.Error() == "" {
			t.Error("expected non-empty error message")
		}
		if !IsNotARepo(err) {
			t.Error("IsNotARepo should return true")
		}
	})

	t.Run("CommitNotFoundError", func(t *testing.T) {
		err := &CommitNotFoundError{SHA: "abc123"}
		if err.Error() == "" {
			t.Error("expected non-empty error message")
		}
		if !IsCommitNotFound(err) {
			t.Error("IsCommitNotFound should return true")
		}
	})
}
