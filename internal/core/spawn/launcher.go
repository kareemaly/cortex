package spawn

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hiveryn/agentruntime"
)

// WriteLauncherScript generates a bash launcher from a LaunchSpec plus
// extra Cortex env vars. The script uses an argv-safe bash array and
// exec so the command is not subject to lossy string-join interpretation.
func WriteLauncherScript(spec agentruntime.LaunchSpec, extraEnv map[string]string, identifier, configDir string) (string, error) {
	if configDir == "" {
		configDir = os.TempDir()
	}

	filename := fmt.Sprintf("cortex-launcher-%s.sh", identifier)
	path := filepath.Join(configDir, filename)

	cleanupFiles := append([]string{path}, spec.CleanupPaths...)

	script := buildLauncherScript(spec, extraEnv, cleanupFiles)

	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		return "", fmt.Errorf("write launcher script: %w", err)
	}

	return path, nil
}

func buildLauncherScript(spec agentruntime.LaunchSpec, extraEnv map[string]string, cleanupFiles []string) string {
	var sb strings.Builder

	sb.WriteString("#!/usr/bin/env bash\n")

	if len(cleanupFiles) > 0 {
		sb.WriteString("trap 'rm -f")
		for _, f := range cleanupFiles {
			sb.WriteString(" ")
			sb.WriteString(shellQuote(f))
		}
		sb.WriteString("' EXIT\n")
	}

	for k, v := range extraEnv {
		sb.WriteString(fmt.Sprintf("export %s=%s\n", k, shellQuote(v)))
	}
	for k, v := range spec.Env {
		if _, overridden := extraEnv[k]; overridden {
			continue
		}
		sb.WriteString(fmt.Sprintf("export %s=%s\n", k, shellQuote(v)))
	}

	sb.WriteString("args=(")
	sb.WriteString(shellQuote(spec.Command))
	for _, arg := range spec.Args {
		sb.WriteString(" ")
		sb.WriteString(shellQuote(arg))
	}
	sb.WriteString(")\n")
	sb.WriteString(`exec "${args[@]}"`)
	sb.WriteString("\n")

	return sb.String()
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
