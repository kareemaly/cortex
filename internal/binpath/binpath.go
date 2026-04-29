package binpath

import (
	"os"
	"os/exec"
	"path/filepath"
)

func findSiblingBinary(names ...string) (string, bool) {
	exe, err := os.Executable()
	if err != nil {
		return "", false
	}

	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", false
	}

	base := filepath.Base(exe)
	if base != "cortex" && base != "cortexd" {
		return "", false
	}

	dir := filepath.Dir(exe)
	for _, name := range names {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}

	return "", false
}

// FindCortex returns the absolute path to the cortex binary.
// It uses the following strategy:
// 1. If current binary is cortex or cortexd, derive path from same directory
// 2. Fall back to exec.LookPath("cortex")
func FindCortex() (string, error) {
	if path, ok := findSiblingBinary("cortex"); ok {
		return path, nil
	}

	return exec.LookPath("cortex")
}

// FindCortexd returns the absolute path to the cortexd binary.
// It uses the following strategy:
// 1. If current binary is cortex or cortexd, derive path from same directory
// 2. Fall back to exec.LookPath("cortexd")
func FindCortexd() (string, error) {
	if path, ok := findSiblingBinary("cortexd"); ok {
		return path, nil
	}

	// Fallback: search PATH
	return exec.LookPath("cortexd")
}
