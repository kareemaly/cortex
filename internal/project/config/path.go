package config

import (
	"os"
	"path/filepath"
	"strings"
)

// ResolvePath resolves extend paths:
// - Absolute: as-is
// - Tilde (~): expand to home directory
// - Relative: resolve from projectRoot
func ResolvePath(path, projectRoot string) (string, error) {
	if path == "" {
		return "", nil
	}

	// Expand tilde to home directory
	if strings.HasPrefix(path, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(homeDir, path[1:])
	}

	// If already absolute, return as-is
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}

	// Resolve relative to projectRoot
	return filepath.Clean(filepath.Join(projectRoot, path)), nil
}

// ValidateExtendPath resolves path and verifies the directory exists.
func ValidateExtendPath(extendPath, projectRoot string) (string, error) {
	if extendPath == "" {
		return "", nil
	}

	resolved, err := ResolvePath(extendPath, projectRoot)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return "", &ExtendPathNotFoundError{
				Path:       extendPath,
				ResolvedTo: resolved,
			}
		}
		return "", err
	}

	if !info.IsDir() {
		return "", &ExtendPathNotFoundError{
			Path:       extendPath,
			ResolvedTo: resolved,
		}
	}

	return resolved, nil
}
