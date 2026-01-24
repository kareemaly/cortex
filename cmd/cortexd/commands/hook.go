package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

const (
	defaultDaemonURL = "http://localhost:4200"
	hookTimeout      = 5 * time.Second
)

// hookInput represents the JSON input from Claude hooks.
type hookInput struct {
	ToolName string `json:"tool_name"`
}

var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Handle Claude hook callbacks",
	Long:  `Subcommands for handling Claude hook events (PostToolUse, Stop, PermissionRequest).`,
}

var hookPostToolUseCmd = &cobra.Command{
	Use:   "post-tool-use",
	Short: "Handle PostToolUse hook",
	Long:  `Called after a tool is used. Updates agent status to in_progress.`,
	RunE:  runHookPostToolUse,
}

var hookStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Handle Stop hook",
	Long:  `Called when agent stops. Updates agent status to idle.`,
	RunE:  runHookStop,
}

var hookPermissionRequestCmd = &cobra.Command{
	Use:   "permission-request",
	Short: "Handle PermissionRequest hook",
	Long:  `Called when agent requests permission. Updates agent status to waiting_permission.`,
	RunE:  runHookPermissionRequest,
}

func init() {
	hookCmd.AddCommand(hookPostToolUseCmd)
	hookCmd.AddCommand(hookStopCmd)
	hookCmd.AddCommand(hookPermissionRequestCmd)
	rootCmd.AddCommand(hookCmd)
}

func runHookPostToolUse(cmd *cobra.Command, args []string) error {
	ticketID := os.Getenv("CORTEX_TICKET_ID")
	projectPath := os.Getenv("CORTEX_PROJECT")
	if ticketID == "" || projectPath == "" {
		// Fail gracefully if env vars not set
		return nil
	}

	// Read tool name from stdin
	var toolName *string
	input, err := io.ReadAll(os.Stdin)
	if err == nil && len(input) > 0 {
		var hookData hookInput
		if json.Unmarshal(input, &hookData) == nil && hookData.ToolName != "" {
			toolName = &hookData.ToolName
		}
	}

	return postAgentStatus(ticketID, projectPath, "in_progress", toolName)
}

func runHookStop(cmd *cobra.Command, args []string) error {
	ticketID := os.Getenv("CORTEX_TICKET_ID")
	projectPath := os.Getenv("CORTEX_PROJECT")
	if ticketID == "" || projectPath == "" {
		return nil
	}

	return postAgentStatus(ticketID, projectPath, "idle", nil)
}

func runHookPermissionRequest(cmd *cobra.Command, args []string) error {
	ticketID := os.Getenv("CORTEX_TICKET_ID")
	projectPath := os.Getenv("CORTEX_PROJECT")
	if ticketID == "" || projectPath == "" {
		return nil
	}

	return postAgentStatus(ticketID, projectPath, "waiting_permission", nil)
}

// postAgentStatus sends a status update to the daemon API.
func postAgentStatus(ticketID, projectPath, status string, tool *string) error {
	payload := map[string]any{
		"ticket_id": ticketID,
		"status":    status,
	}
	if tool != nil {
		payload["tool"] = *tool
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil // Fail gracefully
	}

	url := fmt.Sprintf("%s/agent/status", defaultDaemonURL)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil // Fail gracefully
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cortex-Project", projectPath)

	client := &http.Client{Timeout: hookTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil // Fail gracefully if daemon unreachable
	}
	defer func() { _ = resp.Body.Close() }()

	// We don't return errors here - hooks should fail gracefully
	return nil
}
