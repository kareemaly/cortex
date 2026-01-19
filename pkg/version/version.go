package version

import (
	"fmt"
	"runtime"
)

// These variables are set at build time using ldflags.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// Info contains structured version information.
type Info struct {
	Version   string
	Commit    string
	BuildDate string
	GoVersion string
	Platform  string
}

// Get returns structured version information.
func Get() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
	}
}

// String returns the full version string.
func String(name string) string {
	return fmt.Sprintf("%s %s (commit: %s, built: %s)", name, Version, Commit, BuildDate)
}
