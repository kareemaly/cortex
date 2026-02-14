package install

import (
	"bytes"
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed defaults/*
var defaultsFS embed.FS

// CompareStatus represents the comparison result for a file.
type CompareStatus int

const (
	// CompareUnchanged indicates the file is identical to embedded version.
	CompareUnchanged CompareStatus = iota
	// CompareWillUpdate indicates the file exists but differs from embedded version.
	CompareWillUpdate
	// CompareWillCreate indicates the file does not exist and will be created.
	CompareWillCreate
)

// String returns a human-readable status.
func (s CompareStatus) String() string {
	switch s {
	case CompareUnchanged:
		return "unchanged"
	case CompareWillUpdate:
		return "will update"
	case CompareWillCreate:
		return "will create"
	default:
		return "unknown"
	}
}

// CompareItem represents the comparison result for a single file.
type CompareItem struct {
	Path            string
	Status          CompareStatus
	IsDir           bool
	Error           error
	EmbeddedContent []byte // Content from embedded FS (for diff generation)
	DiskContent     []byte // Content from disk (for diff generation)
}

// CopyEmbeddedDefaults copies embedded default config to target directory.
// Returns a list of SetupItems indicating what was created or skipped.
func CopyEmbeddedDefaults(configName, targetDir string, force bool) ([]SetupItem, error) {
	srcDir := filepath.Join("defaults", configName)
	return copyEmbeddedDir(defaultsFS, srcDir, targetDir, force)
}

// CompareEmbeddedDefaults compares embedded defaults against files on disk.
// Returns a list of CompareItems indicating what would change if upgraded.
func CompareEmbeddedDefaults(configName, targetDir string) ([]CompareItem, error) {
	srcDir := filepath.Join("defaults", configName)
	return compareEmbeddedDir(defaultsFS, srcDir, targetDir)
}

// compareEmbeddedDir recursively compares embedded directory to disk.
func compareEmbeddedDir(embedFS embed.FS, srcDir, dstDir string) ([]CompareItem, error) {
	var items []CompareItem

	err := fs.WalkDir(embedFS, srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path from srcDir
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		dstPath := filepath.Join(dstDir, relPath)

		if d.IsDir() {
			// Check if directory exists
			info, err := os.Stat(dstPath)
			item := CompareItem{Path: dstPath, IsDir: true}
			if err != nil {
				if os.IsNotExist(err) {
					item.Status = CompareWillCreate
				} else {
					item.Error = err
				}
			} else if !info.IsDir() {
				item.Error = &PathNotDirectoryError{Path: dstPath}
			} else {
				item.Status = CompareUnchanged
			}
			items = append(items, item)
			return nil
		}

		// It's a file - read content from embedded FS
		embeddedContent, err := embedFS.ReadFile(path)
		if err != nil {
			return err
		}

		item := CompareItem{Path: dstPath, IsDir: false}

		// Check if file exists on disk
		diskContent, err := os.ReadFile(dstPath)
		if err != nil {
			if os.IsNotExist(err) {
				item.Status = CompareWillCreate
				item.EmbeddedContent = embeddedContent
			} else {
				item.Error = err
			}
		} else {
			// Compare contents
			if bytes.Equal(embeddedContent, diskContent) {
				item.Status = CompareUnchanged
			} else {
				item.Status = CompareWillUpdate
				item.EmbeddedContent = embeddedContent
				item.DiskContent = diskContent
			}
		}

		items = append(items, item)
		return nil
	})

	if err != nil {
		return items, err
	}

	return items, nil
}

// copyEmbeddedDir recursively copies embedded directory to disk.
// Skips files that exist unless force=true.
func copyEmbeddedDir(embedFS embed.FS, srcDir, dstDir string, force bool) ([]SetupItem, error) {
	var items []SetupItem

	// Create the target directory first
	item := ensureDir(dstDir)
	items = append(items, item)
	if item.Error != nil {
		return items, item.Error
	}

	err := fs.WalkDir(embedFS, srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path from srcDir
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		// Skip the root directory itself (already created above)
		if relPath == "." {
			return nil
		}

		dstPath := filepath.Join(dstDir, relPath)

		if d.IsDir() {
			item := ensureDir(dstPath)
			items = append(items, item)
			if item.Error != nil {
				return item.Error
			}
			return nil
		}

		// It's a file - read content from embedded FS
		content, err := embedFS.ReadFile(path)
		if err != nil {
			return err
		}

		item := ensureConfigFile(dstPath, string(content), force)
		items = append(items, item)
		if item.Error != nil {
			return item.Error
		}

		return nil
	})

	if err != nil {
		return items, err
	}

	return items, nil
}
