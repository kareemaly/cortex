package git

import (
	"strconv"
	"strings"
)

// DiffStats contains summary statistics for a diff.
type DiffStats struct {
	FilesChanged int
	Insertions   int
	Deletions    int
}

// ChangedFile represents a file changed between commits.
type ChangedFile struct {
	Path   string
	Status string // "A" added, "M" modified, "D" deleted, "R" renamed, "C" copied
}

// GetDiffStats returns diff statistics between a base commit and HEAD.
func GetDiffStats(repoPath, baseCommit string) (*DiffStats, error) {
	return GetDiffStatsBetween(repoPath, baseCommit, "HEAD")
}

// GetDiffStatsBetween returns diff statistics between two refs.
func GetDiffStatsBetween(repoPath, fromRef, toRef string) (*DiffStats, error) {
	output, err := runGit(repoPath, "diff", "--numstat", fromRef+".."+toRef)
	if err != nil {
		if isNotARepoError(err) {
			return nil, &NotARepoError{Path: repoPath}
		}
		if isUnknownRevisionError(err) {
			return nil, &CommitNotFoundError{SHA: fromRef}
		}
		return nil, err
	}

	stats := &DiffStats{}
	if output == "" {
		return stats, nil
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		// Handle binary files (shown as "-" for insertions/deletions)
		if fields[0] != "-" {
			if add, err := strconv.Atoi(fields[0]); err == nil {
				stats.Insertions += add
			}
		}
		if fields[1] != "-" {
			if del, err := strconv.Atoi(fields[1]); err == nil {
				stats.Deletions += del
			}
		}
		stats.FilesChanged++
	}

	return stats, nil
}

// GetChangedFiles returns the list of files changed between a base commit and HEAD.
func GetChangedFiles(repoPath, baseCommit string) ([]ChangedFile, error) {
	return GetChangedFilesBetween(repoPath, baseCommit, "HEAD")
}

// GetChangedFilesBetween returns the list of files changed between two refs.
func GetChangedFilesBetween(repoPath, fromRef, toRef string) ([]ChangedFile, error) {
	output, err := runGit(repoPath, "diff", "--name-status", fromRef+".."+toRef)
	if err != nil {
		if isNotARepoError(err) {
			return nil, &NotARepoError{Path: repoPath}
		}
		if isUnknownRevisionError(err) {
			return nil, &CommitNotFoundError{SHA: fromRef}
		}
		return nil, err
	}

	if output == "" {
		return nil, nil
	}

	lines := strings.Split(output, "\n")
	var files []ChangedFile

	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		status := fields[0]
		path := fields[1]

		// For renames (R100) and copies (C100), take the destination path
		if len(fields) >= 3 && (strings.HasPrefix(status, "R") || strings.HasPrefix(status, "C")) {
			path = fields[2]
			// Normalize to single letter
			status = string(status[0])
		}

		files = append(files, ChangedFile{
			Path:   path,
			Status: status,
		})
	}

	return files, nil
}
