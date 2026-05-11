package api

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	architectconfig "github.com/kareemaly/cortex/internal/architect/config"
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

func resolveGitRepoDir(repoDir string) (string, error) {
	if repoDir == "" {
		return "", fmt.Errorf("repo path is empty")
	}

	dir := expandHome(repoDir)
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve repo path: %w", err)
	}

	info, err := os.Stat(absDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("repo directory does not exist: %s", absDir)
		}
		return "", fmt.Errorf("stat repo directory: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("repo path is not a directory: %s", absDir)
	}

	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = absDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("repo is not a git repository: %s", absDir)
	}
	if strings.TrimSpace(string(out)) != "true" {
		return "", fmt.Errorf("repo is not a git repository: %s", absDir)
	}

	return absDir, nil
}

func resolveTicketRepoDir(projectPath, repoKey string) (string, error) {
	if repoKey == "" {
		return "", fmt.Errorf("ticket repo key is empty")
	}

	cfg, err := architectconfig.Load(projectPath)
	if err != nil {
		return "", fmt.Errorf("load project config: %w", err)
	}

	repoPath, err := cfg.ResolveRepoPath(repoKey)
	if err != nil {
		return "", err
	}

	return resolveGitRepoDir(repoPath)
}

type gitCommitMeta struct {
	SHA         string
	Subject     string
	AuthorName  string
	AuthorEmail string
	AuthoredAt  time.Time
}

type gitFileChange struct {
	Path      string
	OldPath   *string
	Status    string
	IsBinary  bool
	Additions int
	Deletions int
	Patch     string
	Before    *string
	After     *string
}

func runGit(repoDir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return out, nil
}

func gitCommitMetadata(repoDir, sha string) (*gitCommitMeta, error) {
	out, err := runGit(repoDir, "show", "-s", "--format=%H%x00%s%x00%an%x00%ae%x00%aI", sha)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(strings.TrimRight(string(out), "\n"), "\x00")
	if len(parts) != 5 {
		return nil, fmt.Errorf("unexpected git show metadata output for %s", sha)
	}
	authoredAt, err := time.Parse(time.RFC3339, parts[4])
	if err != nil {
		return nil, fmt.Errorf("parse authored_at for %s: %w", sha, err)
	}
	return &gitCommitMeta{
		SHA:         parts[0],
		Subject:     parts[1],
		AuthorName:  parts[2],
		AuthorEmail: parts[3],
		AuthoredAt:  authoredAt,
	}, nil
}

func gitCommitParent(repoDir, sha string) (string, error) {
	out, err := runGit(repoDir, "rev-list", "--parents", "-n", "1", sha)
	if err != nil {
		return "", err
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) <= 1 {
		return "", nil
	}
	return fields[1], nil
}

func gitNameStatus(repoDir, sha string) ([]gitFileChange, error) {
	out, err := runGit(repoDir, "diff-tree", "--root", "--find-renames", "--find-copies", "--no-commit-id", "--name-status", "-z", "-r", sha)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return []gitFileChange{}, nil
	}

	parts := bytes.Split(out, []byte{0})
	parts = parts[:len(parts)-1]
	changes := make([]gitFileChange, 0)
	for i := 0; i < len(parts); {
		if len(parts[i]) == 0 {
			i++
			continue
		}
		code := string(parts[i])
		i++

		switch {
		case strings.HasPrefix(code, "R") || strings.HasPrefix(code, "C"):
			if i+1 >= len(parts) {
				return nil, fmt.Errorf("unexpected rename/copy output for commit %s", sha)
			}
			oldPath := string(parts[i])
			newPath := string(parts[i+1])
			i += 2
			status := "renamed"
			if strings.HasPrefix(code, "C") {
				status = "copied"
			}
			changes = append(changes, gitFileChange{
				Path:    newPath,
				OldPath: &oldPath,
				Status:  status,
			})
		default:
			if i >= len(parts) {
				return nil, fmt.Errorf("unexpected name-status output for commit %s", sha)
			}
			path := string(parts[i])
			i++
			changes = append(changes, gitFileChange{
				Path:   path,
				Status: diffStatus(code),
			})
		}
	}
	return changes, nil
}

func gitNumstat(repoDir, sha string) ([][2]string, error) {
	out, err := runGit(repoDir, "diff-tree", "--root", "--find-renames", "--find-copies", "--no-commit-id", "--numstat", "-z", "-r", sha)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return [][2]string{}, nil
	}

	parts := bytes.Split(out, []byte{0})
	parts = parts[:len(parts)-1]
	stats := make([][2]string, 0)
	for _, part := range parts {
		fields := strings.Split(string(part), "\t")
		if len(fields) < 3 {
			return nil, fmt.Errorf("unexpected numstat output for commit %s", sha)
		}
		stats = append(stats, [2]string{fields[0], fields[1]})
	}
	return stats, nil
}

func gitFilePatch(repoDir, sha, path string) (string, error) {
	out, err := runGit(repoDir, "show", "--format=", "--patch", "--no-color", "--find-renames", "--find-copies", sha, "--", path)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func gitFileContent(repoDir, spec string) (*string, bool, error) {
	out, err := runGit(repoDir, "show", spec)
	if err != nil {
		return nil, false, err
	}
	if !utf8.Valid(out) {
		return nil, true, nil
	}
	content := string(out)
	return &content, false, nil
}

func buildCommitDiff(repoDir, sha string) (*CommitDiffResponse, error) {
	meta, err := gitCommitMetadata(repoDir, sha)
	if err != nil {
		return nil, err
	}
	parent, err := gitCommitParent(repoDir, sha)
	if err != nil {
		return nil, err
	}
	changes, err := gitNameStatus(repoDir, sha)
	if err != nil {
		return nil, err
	}
	stats, err := gitNumstat(repoDir, sha)
	if err != nil {
		return nil, err
	}
	if len(changes) != len(stats) {
		return nil, fmt.Errorf("git diff metadata mismatch for commit %s", sha)
	}

	files := make([]DiffFileResponse, 0, len(changes))
	for i, change := range changes {
		additions, deletions, isBinary, err := parseNumstat(stats[i][0], stats[i][1])
		if err != nil {
			return nil, err
		}
		patch, err := gitFilePatch(repoDir, sha, change.Path)
		if err != nil {
			return nil, err
		}

		before, after := (*string)(nil), (*string)(nil)
		if change.Status == "added" {
			after, isBinary, err = gitFileContent(repoDir, sha+":"+change.Path)
			if err != nil {
				return nil, err
			}
		}
		if change.Status == "deleted" && parent != "" {
			before, isBinary, err = gitFileContent(repoDir, parent+":"+change.Path)
			if err != nil {
				return nil, err
			}
		}

		files = append(files, DiffFileResponse{
			Path:      change.Path,
			OldPath:   change.OldPath,
			Status:    change.Status,
			IsBinary:  isBinary || change.IsBinary,
			Additions: additions,
			Deletions: deletions,
			Patch:     patch,
			Before:    before,
			After:     after,
		})
	}

	return &CommitDiffResponse{
		SHA:         meta.SHA,
		Subject:     meta.Subject,
		AuthorName:  meta.AuthorName,
		AuthorEmail: meta.AuthorEmail,
		AuthoredAt:  meta.AuthoredAt,
		Files:       files,
	}, nil
}

func diffStatus(code string) string {
	switch {
	case strings.HasPrefix(code, "A"):
		return "added"
	case strings.HasPrefix(code, "D"):
		return "deleted"
	case strings.HasPrefix(code, "R"):
		return "renamed"
	case strings.HasPrefix(code, "C"):
		return "copied"
	default:
		return "modified"
	}
}

func parseNumstat(adds, dels string) (int, int, bool, error) {
	if adds == "-" || dels == "-" {
		return 0, 0, true, nil
	}
	additions, err := strconv.Atoi(adds)
	if err != nil {
		return 0, 0, false, fmt.Errorf("parse additions %q: %w", adds, err)
	}
	deletions, err := strconv.Atoi(dels)
	if err != nil {
		return 0, 0, false, fmt.Errorf("parse deletions %q: %w", dels, err)
	}
	return additions, deletions, false, nil
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
