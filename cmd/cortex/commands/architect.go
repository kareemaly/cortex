package commands

import (
	"fmt"
	"os"
	"path/filepath"

	architectconfig "github.com/kareemaly/cortex/internal/architect/config"
	daemonconfig "github.com/kareemaly/cortex/internal/daemon/config"
	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/spf13/cobra"
)

var architectCmd = &cobra.Command{
	Use:   "architect",
	Short: "Manage architect workspaces",
}

func init() {
	rootCmd.AddCommand(architectCmd)
}

// resolveArchitectPath resolves an architect path from name or current directory.
// If name is provided, looks up in global config by title or directory name.
// If name is empty, finds architect root from current working directory.
func resolveArchitectPath(name string) (string, error) {
	if name != "" {
		cfg, err := daemonconfig.Load()
		if err != nil {
			return "", fmt.Errorf("failed to load global config: %w", err)
		}
		for _, a := range cfg.Architects {
			if a.Title == name || filepath.Base(a.Path) == name {
				return a.Path, nil
			}
		}
		return "", fmt.Errorf("architect %q not found", name)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	root, err := architectconfig.FindArchitectRoot(cwd)
	if err != nil {
		return "", fmt.Errorf("not in a cortex architect (no cortex.yaml found)")
	}
	return root, nil
}

// newProjectClient creates an SDK client scoped to the architect path.
func newProjectClient(architectPath string) *sdk.Client {
	return sdk.DefaultClient(architectPath)
}
