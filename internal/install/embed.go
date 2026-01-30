package install

import (
	"embed"
	"io/fs"
	"path/filepath"
)

//go:embed defaults/*
var defaultsFS embed.FS

// copyEmbeddedDefaults copies embedded default config to target directory.
// Returns a list of SetupItems indicating what was created or skipped.
func copyEmbeddedDefaults(configName, targetDir string, force bool) ([]SetupItem, error) {
	srcDir := filepath.Join("defaults", configName)
	return copyEmbeddedDir(defaultsFS, srcDir, targetDir, force)
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
