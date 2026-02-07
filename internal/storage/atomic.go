package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

// AtomicWriteFile writes data to target atomically using a temp file + rename.
// The temp file is created in the same directory as target for same-filesystem rename.
func AtomicWriteFile(target string, data []byte) error {
	dir := filepath.Dir(target)

	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	defer func() {
		if tmpPath != "" {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, target); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}

	tmpPath = "" // prevent deferred cleanup
	return nil
}
