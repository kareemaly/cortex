package binpath

import (
	"os"
	"os/exec"
	"path/filepath"
)

// FindCortexd returns the absolute path to the cortexd binary.
// It uses the following strategy:
// 1. If current binary is cortex or cortexd, derive path from same directory
// 2. Fall back to exec.LookPath("cortexd")
func FindCortexd() (string, error) {
	// Get current executable path
	exe, err := os.Executable()
	if err == nil {
		exe, err = filepath.EvalSymlinks(exe)
		if err == nil {
			base := filepath.Base(exe)
			dir := filepath.Dir(exe)

			// If running as cortex or cortexd, derive path
			if base == "cortex" || base == "cortexd" {
				cortexdPath := filepath.Join(dir, "cortexd")
				if _, err := os.Stat(cortexdPath); err == nil {
					return cortexdPath, nil
				}
			}
		}
	}

	// Fallback: search PATH
	return exec.LookPath("cortexd")
}
