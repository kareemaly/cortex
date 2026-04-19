package spawn

import (
	"fmt"
	"os"
	"path/filepath"
)

// GenerateOpenCodeStatusPlugin returns a TypeScript plugin string that
// appends JSONL status events to statusFilePath. The cortex daemon reads
// that file via the opencode supervisor and forwards events to /agent/status.
// The plugin emits only canonical status names (working, idle, awaiting_input,
// error) — no wire-format translation happens on the Go side.
//
// The plugin runs in Bun inside opencode; fs calls use node:fs. Values are
// baked in via string interpolation (no env var lookups at runtime).
func GenerateOpenCodeStatusPlugin(statusFilePath string) string {
	return fmt.Sprintf(`// Cortex status plugin — auto-generated, do not edit.
// Appends JSONL status events to a file the cortex daemon tails.

import { appendFileSync } from "node:fs";

const STATUS_FILE = %q;

function emit(status: string, tool?: string) {
  const payload: Record<string, string> = { status };
  if (tool) payload.tool = tool;
  try {
    appendFileSync(STATUS_FILE, JSON.stringify(payload) + "\n");
  } catch {
    // Fire-and-forget: the file may have been cleaned up already.
  }
}

export default async (_input: any) => ({
  event: async ({ event }: { event: { type: string; properties?: Record<string, any> } }) => {
    switch (event.type) {
      case "session.status": {
        const s = event.properties?.status?.type;
        if (s === "busy") emit("working");
        else if (s === "idle") emit("idle");
        else if (s === "retry") emit("error");
        break;
      }
      case "permission.asked":
        emit("awaiting_input");
        break;
      case "permission.replied":
        emit("working");
        break;
    }
  },
  "tool.execute.before": async (hookInput: any) => {
    emit("working", hookInput?.tool as string | undefined);
  },
  "tool.execute.after": async () => {
    emit("working");
  },
});
`, statusFilePath)
}

// WriteOpenCodePluginDir creates a temporary directory with the status plugin
// written to plugin/cortex-status.ts. The plugin/ subdirectory is made
// read-only (0555) to prevent OpenCode from running a dependency auto-install.
// Returns the temp dir path (used as OPENCODE_CONFIG_DIR).
func WriteOpenCodePluginDir(pluginContent, identifier string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "cortex-opencode-")
	if err != nil {
		return "", fmt.Errorf("create opencode plugin temp dir: %w", err)
	}

	pluginDir := filepath.Join(tmpDir, "plugin")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", fmt.Errorf("create plugin subdirectory: %w", err)
	}

	pluginPath := filepath.Join(pluginDir, "cortex-status.ts")
	if err := os.WriteFile(pluginPath, []byte(pluginContent), 0644); err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", fmt.Errorf("write opencode status plugin: %w", err)
	}

	// Make plugin dir read-only so OpenCode skips dependency auto-install.
	if err := os.Chmod(pluginDir, 0555); err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", fmt.Errorf("chmod plugin directory: %w", err)
	}

	return tmpDir, nil
}

// OpenCodeStatusFilePath returns the JSONL file path for an opencode status
// tailer. The cortex plugin appends to this file; the daemon tailer reads it.
// configDir defaults to os.TempDir() when empty.
func OpenCodeStatusFilePath(identifier, configDir string) string {
	if configDir == "" {
		configDir = os.TempDir()
	}
	return filepath.Join(configDir, fmt.Sprintf("cortex-opencode-status-%s.jsonl", identifier))
}
