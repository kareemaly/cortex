package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	daemonconfig "github.com/kareemaly/cortex/internal/daemon/config"
	"github.com/spf13/cobra"
)

const hookTimeout = 5 * time.Second

// hookInput represents the JSON input from Claude hooks.
type hookInput struct {
	ToolName    string `json:"tool_name"`
	Reason      string `json:"reason"`
	Error       string `json:"error"`
	IsInterrupt bool   `json:"is_interrupt"`
	AgentType   string `json:"agent_type"`
	AgentID     string `json:"agent_id"`
	Source      string `json:"source"`
}

var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Handle Claude hook callbacks",
	Long:  `Subcommands for handling Claude hook events.`,
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

var hookSessionStartCmd = &cobra.Command{
	Use:   "session-start",
	Short: "Handle SessionStart hook",
	Long:  `Called when the agent session starts. Transitions from starting to in_progress.`,
	RunE:  runHookSessionStart,
}

var hookSessionEndCmd = &cobra.Command{
	Use:   "session-end",
	Short: "Handle SessionEnd hook",
	Long:  `Called when the agent session ends. Sets status to idle or error based on reason.`,
	RunE:  runHookSessionEnd,
}

var hookPostToolUseFailureCmd = &cobra.Command{
	Use:   "post-tool-use-failure",
	Short: "Handle PostToolUseFailure hook",
	Long:  `Called after a tool use fails. Updates status with error context.`,
	RunE:  runHookPostToolUseFailure,
}

var hookSubagentStartCmd = &cobra.Command{
	Use:   "subagent-start",
	Short: "Handle SubagentStart hook",
	Long:  `Called when a subagent is spawned. Updates tool to reflect subagent type.`,
	RunE:  runHookSubagentStart,
}

var hookSubagentStopCmd = &cobra.Command{
	Use:   "subagent-stop",
	Short: "Handle SubagentStop hook",
	Long:  `Called when a subagent completes. Clears tool as main agent resumes.`,
	RunE:  runHookSubagentStop,
}

func init() {
	hookCmd.AddCommand(hookPostToolUseCmd)
	hookCmd.AddCommand(hookStopCmd)
	hookCmd.AddCommand(hookPermissionRequestCmd)
	hookCmd.AddCommand(hookSessionStartCmd)
	hookCmd.AddCommand(hookSessionEndCmd)
	hookCmd.AddCommand(hookPostToolUseFailureCmd)
	hookCmd.AddCommand(hookSubagentStartCmd)
	hookCmd.AddCommand(hookSubagentStopCmd)
	rootCmd.AddCommand(hookCmd)
}

// readHookInput reads and parses the JSON hook payload from stdin.
func readHookInput() *hookInput {
	input, err := io.ReadAll(os.Stdin)
	if err != nil || len(input) == 0 {
		return &hookInput{}
	}
	var data hookInput
	if json.Unmarshal(input, &data) != nil {
		return &hookInput{}
	}
	return &data
}

func runHookPostToolUse(cmd *cobra.Command, args []string) error {
	ticketID := os.Getenv("CORTEX_TICKET_ID")
	projectPath := os.Getenv("CORTEX_PROJECT")
	if ticketID == "" || projectPath == "" {
		return nil
	}

	data := readHookInput()
	var toolName *string
	if data.ToolName != "" {
		toolName = &data.ToolName
	}

	return postAgentStatus(ticketID, projectPath, "in_progress", toolName, nil)
}

func runHookStop(cmd *cobra.Command, args []string) error {
	ticketID := os.Getenv("CORTEX_TICKET_ID")
	projectPath := os.Getenv("CORTEX_PROJECT")
	if ticketID == "" || projectPath == "" {
		return nil
	}

	return postAgentStatus(ticketID, projectPath, "idle", nil, nil)
}

func runHookPermissionRequest(cmd *cobra.Command, args []string) error {
	ticketID := os.Getenv("CORTEX_TICKET_ID")
	projectPath := os.Getenv("CORTEX_PROJECT")
	if ticketID == "" || projectPath == "" {
		return nil
	}

	return postAgentStatus(ticketID, projectPath, "waiting_permission", nil, nil)
}

func runHookSessionStart(cmd *cobra.Command, args []string) error {
	ticketID := os.Getenv("CORTEX_TICKET_ID")
	projectPath := os.Getenv("CORTEX_PROJECT")
	if ticketID == "" || projectPath == "" {
		return nil
	}

	return postAgentStatus(ticketID, projectPath, "in_progress", nil, nil)
}

func runHookSessionEnd(cmd *cobra.Command, args []string) error {
	ticketID := os.Getenv("CORTEX_TICKET_ID")
	projectPath := os.Getenv("CORTEX_PROJECT")
	if ticketID == "" || projectPath == "" {
		return nil
	}

	data := readHookInput()

	// Normal exits → idle, abnormal exits → error with reason in work
	if data.Reason == "prompt_input_exit" || data.Reason == "clear" {
		return postAgentStatus(ticketID, projectPath, "idle", nil, nil)
	}

	var work *string
	if data.Reason != "" {
		work = &data.Reason
	}
	return postAgentStatus(ticketID, projectPath, "error", nil, work)
}

func runHookPostToolUseFailure(cmd *cobra.Command, args []string) error {
	ticketID := os.Getenv("CORTEX_TICKET_ID")
	projectPath := os.Getenv("CORTEX_PROJECT")
	if ticketID == "" || projectPath == "" {
		return nil
	}

	data := readHookInput()
	var toolName *string
	if data.ToolName != "" {
		toolName = &data.ToolName
	}
	var work *string
	if data.Error != "" {
		work = &data.Error
	}

	return postAgentStatus(ticketID, projectPath, "in_progress", toolName, work)
}

func runHookSubagentStart(cmd *cobra.Command, args []string) error {
	ticketID := os.Getenv("CORTEX_TICKET_ID")
	projectPath := os.Getenv("CORTEX_PROJECT")
	if ticketID == "" || projectPath == "" {
		return nil
	}

	data := readHookInput()
	toolLabel := "Task"
	if data.AgentType != "" {
		toolLabel = fmt.Sprintf("Task (%s)", data.AgentType)
	}

	return postAgentStatus(ticketID, projectPath, "in_progress", &toolLabel, nil)
}

func runHookSubagentStop(cmd *cobra.Command, args []string) error {
	ticketID := os.Getenv("CORTEX_TICKET_ID")
	projectPath := os.Getenv("CORTEX_PROJECT")
	if ticketID == "" || projectPath == "" {
		return nil
	}

	return postAgentStatus(ticketID, projectPath, "in_progress", nil, nil)
}

// postAgentStatus sends a status update to the daemon API.
func postAgentStatus(ticketID, projectPath, status string, tool *string, work *string) error {
	payload := map[string]any{
		"ticket_id": ticketID,
		"status":    status,
	}
	if tool != nil {
		payload["tool"] = *tool
	}
	if work != nil {
		payload["work"] = *work
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil // Fail gracefully
	}

	baseURL := os.Getenv("CORTEX_DAEMON_URL")
	if baseURL == "" {
		baseURL = daemonconfig.DefaultDaemonURL
	}
	url := fmt.Sprintf("%s/agent/status", baseURL)
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
