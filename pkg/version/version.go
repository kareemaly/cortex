package version

import "fmt"

// These variables are set at build time using ldflags.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// String returns the full version string.
func String(name string) string {
	return fmt.Sprintf("%s %s (commit: %s, built: %s)", name, Version, Commit, BuildDate)
}
