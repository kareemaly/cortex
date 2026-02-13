package spawn

import (
	"fmt"
	"os"
	"path/filepath"
)

// GenerateOpenCodeStatusPlugin returns a TypeScript plugin string that pushes
// agent status updates to the Cortex daemon's POST /agent/status endpoint.
// Values are baked in via string interpolation because the plugin runs in Bun,
// not a shell (no env var expansion).
func GenerateOpenCodeStatusPlugin(daemonURL, ticketID, projectPath string) string {
	return fmt.Sprintf(`// Cortex status plugin — auto-generated, do not edit.
// Pushes OpenCode agent status to the Cortex daemon.

const DAEMON_URL = %q;
const TICKET_ID = %q;
const PROJECT_PATH = %q;

function send(status: string, tool?: string) {
  const body: Record<string, string> = { ticket_id: TICKET_ID, status };
  if (tool) body.tool = tool;
  fetch(DAEMON_URL + "/agent/status", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-Cortex-Project": PROJECT_PATH,
    },
    body: JSON.stringify(body),
    signal: AbortSignal.timeout(5000),
  }).catch(() => {});
}

export default async () => ({
  event: async ({ event }: { event: { type: string; properties?: Record<string, any>; input?: Record<string, any> } }) => {
    switch (event.type) {
      case "session.status": {
        const s = event.properties?.status;
        if (s === "busy") send("in_progress");
        else if (s === "idle") send("idle");
        else if (s === "retry") send("error");
        break;
      }
      case "session.idle":
        send("idle");
        break;
      case "permission.asked":
        send("waiting_permission");
        break;
      case "permission.replied":
        send("in_progress");
        break;
      case "tool.execute.before":
        send("in_progress", event.input?.tool as string | undefined);
        break;
      case "tool.execute.after":
        send("in_progress");
        break;
    }
  },
});
`, daemonURL, ticketID, projectPath)
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
