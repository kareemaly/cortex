package install

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kareemaly/cortex/internal/prompt"
	"github.com/kareemaly/cortex/internal/storage"
)

// agentsMDVars holds template variables for the AGENTS_MD.md prompt.
type agentsMDVars struct {
	ArchitectRoot string
	Repos         []string
	OutputFile    string
}

// AgentProcess holds the running agent process and its stdout scanner.
type AgentProcess struct {
	Cmd     *exec.Cmd
	Scanner *bufio.Scanner
	Agent   string
}

// Wait waits for the agent process to complete.
func (p *AgentProcess) Wait() error {
	return p.Cmd.Wait()
}

// Kill terminates the agent process.
func (p *AgentProcess) Kill() {
	if p.Cmd != nil && p.Cmd.Process != nil {
		_ = p.Cmd.Process.Kill()
	}
}

// AgentEvent represents a parsed event from the agent's JSON stream.
type AgentEvent struct {
	Type     string // "tool", "subagent", "text", "reasoning", "step_finish", "result", "done", "error"
	Tool     string
	Status   string
	Content  string
	Subagent *AgentEventSubagent
	Cost     float64
	Tokens   AgentEventTokens
	Err      error
}

// AgentEventSubagent holds subagent information from a tool event.
type AgentEventSubagent struct {
	AgentType   string
	Description string
}

// AgentEventTokens holds token usage information.
type AgentEventTokens struct {
	Input  int64
	Output int64
}

// StartAgentsMDProcess starts the agent process for AGENTS.md generation and
// returns a handle with a stdout scanner for reading JSON events. The caller
// is responsible for reading from Scanner, calling Wait(), and handling the
// process lifecycle. No output is written to os.Stdout or os.Stderr.
func StartAgentsMDProcess(architectRoot string, repos []string, agent, model string) (*AgentProcess, error) {
	outputFile := "AGENTS.md"
	if agent == "claude" {
		outputFile = "CLAUDE.md"
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	templatePath := filepath.Join(homeDir, ".cortex", "defaults", "main", "prompts", "AGENTS_MD.md")
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read AGENTS_MD.md template: %w", err)
	}

	vars := agentsMDVars{
		ArchitectRoot: architectRoot,
		Repos:         repos,
		OutputFile:    outputFile,
	}

	promptText, err := prompt.RenderTemplate(string(templateContent), vars)
	if err != nil {
		return nil, fmt.Errorf("failed to render template: %w", err)
	}

	switch agent {
	case "claude":
		return startClaudeProcess(repos, architectRoot, promptText)
	case "opencode":
		return startOpenCodeProcess(repos, architectRoot, promptText, model)
	default:
		return nil, fmt.Errorf("unsupported agent: %s", agent)
	}
}

func startClaudeProcess(repos []string, architectRoot, promptText string) (*AgentProcess, error) {
	args := []string{"-p", "--dangerously-skip-permissions", "--output-format", "stream-json", "--verbose"}
	for _, repo := range repos {
		expanded := storage.ExpandHome(repo)
		args = append(args, "--add-dir", expanded)
	}
	args = append(args, "--add-dir", architectRoot)
	args = append(args, promptText)

	cmd := exec.Command("claude", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	// Discard stderr so it doesn't leak to the terminal.
	cmd.Stderr = io.Discard

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("claude failed to start: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer for large JSON lines

	return &AgentProcess{
		Cmd:     cmd,
		Scanner: scanner,
		Agent:   "claude",
	}, nil
}

func startOpenCodeProcess(repos []string, architectRoot, promptText, model string) (*AgentProcess, error) {
	perms := make(map[string]string)
	for _, repo := range repos {
		expanded := storage.ExpandHome(repo)
		perms[expanded] = "allow"
		perms[expanded+"/*"] = "allow"
	}
	architectExpanded := storage.ExpandHome(architectRoot)
	perms[architectExpanded] = "allow"
	perms[architectExpanded+"/*"] = "allow"

	permJSON, err := json.Marshal(map[string]interface{}{"external_directory": perms})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal permissions: %w", err)
	}

	args := []string{"run", "--format", "json"}
	if model != "" {
		args = append(args, "--model", model)
	}

	cmd := exec.Command("opencode", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("OPENCODE_PERMISSION=%s", string(permJSON)))
	cmd.Stdin = strings.NewReader(promptText)
	cmd.Stderr = io.Discard

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("opencode failed to start: %w", err)
	}

	return &AgentProcess{
		Cmd:     cmd,
		Scanner: bufio.NewScanner(stdout),
		Agent:   "opencode",
	}, nil
}

// ReadNextEvent reads the next JSON line from the scanner and parses it into
// an AgentEvent. Returns an event with Type=="done" when the stream ends.
// This function blocks until a meaningful event is read.
func ReadNextEvent(proc *AgentProcess) AgentEvent {
	for {
		if !proc.Scanner.Scan() {
			if err := proc.Scanner.Err(); err != nil {
				return AgentEvent{Type: "error", Err: err}
			}
			return AgentEvent{Type: "done"}
		}

		line := proc.Scanner.Text()
		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue // skip non-JSON lines
		}

		if proc.Agent == "claude" {
			if ev, ok := parseClaudeEvent(raw); ok {
				return ev
			}
		} else {
			if ev, ok := parseOpenCodeEvent(raw); ok {
				return ev
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Claude stream-json event parsing
// ---------------------------------------------------------------------------

func parseClaudeEvent(raw map[string]interface{}) (AgentEvent, bool) {
	eventType, _ := raw["type"].(string)

	switch eventType {
	case "assistant":
		return parseClaudeAssistantEvent(raw)
	case "result":
		return parseClaudeResultEvent(raw)
	}

	return AgentEvent{}, false
}

func parseClaudeAssistantEvent(raw map[string]interface{}) (AgentEvent, bool) {
	msg, _ := raw["message"].(map[string]interface{})
	if msg == nil {
		return AgentEvent{}, false
	}

	content, _ := msg["content"].([]interface{})
	if len(content) == 0 {
		return AgentEvent{}, false
	}

	// We care about the last content block in the array.
	lastBlock, _ := content[len(content)-1].(map[string]interface{})
	if lastBlock == nil {
		return AgentEvent{}, false
	}

	blockType, _ := lastBlock["type"].(string)
	switch blockType {
	case "text":
		text, _ := lastBlock["text"].(string)
		if text == "" {
			return AgentEvent{}, false
		}
		return AgentEvent{Type: "text", Content: text}, true

	case "thinking":
		thinking, _ := lastBlock["thinking"].(string)
		if thinking == "" {
			return AgentEvent{}, false
		}
		return AgentEvent{Type: "reasoning", Content: thinking}, true

	case "tool_use":
		name, _ := lastBlock["name"].(string)
		input, _ := lastBlock["input"].(map[string]interface{})

		var desc string
		if input != nil {
			desc = extractToolContent(name, input)
		}

		return AgentEvent{
			Type:    "tool",
			Tool:    name,
			Status:  "running",
			Content: desc,
		}, true

	case "tool_result":
		return AgentEvent{
			Type:   "tool",
			Status: "completed",
		}, true
	}

	return AgentEvent{}, false
}

func parseClaudeResultEvent(raw map[string]interface{}) (AgentEvent, bool) {
	cost, _ := raw["total_cost_usd"].(float64)
	usage, _ := raw["usage"].(map[string]interface{})

	var tokens AgentEventTokens
	if usage != nil {
		if v, ok := usage["input_tokens"].(float64); ok {
			tokens.Input = int64(v)
		}
		if v, ok := usage["output_tokens"].(float64); ok {
			tokens.Output = int64(v)
		}
	}

	return AgentEvent{
		Type:   "result",
		Cost:   cost,
		Tokens: tokens,
	}, true
}

// ---------------------------------------------------------------------------
// OpenCode JSON event parsing
// ---------------------------------------------------------------------------

func parseOpenCodeEvent(raw map[string]interface{}) (AgentEvent, bool) {
	eventType, _ := raw["type"].(string)
	part, _ := raw["part"].(map[string]interface{})

	switch eventType {
	case "tool_use":
		return parseOpenCodeToolEvent(part)
	case "text":
		return parseOpenCodeTextEvent(part)
	case "reasoning":
		return parseOpenCodeReasoningEvent(part)
	case "step_finish":
		return parseOpenCodeStepFinishEvent(part)
	}

	return AgentEvent{}, false
}

func parseOpenCodeToolEvent(part map[string]interface{}) (AgentEvent, bool) {
	if part == nil {
		return AgentEvent{}, false
	}

	tool, _ := part["tool"].(string)
	state, _ := part["state"].(map[string]interface{})
	if state == nil {
		return AgentEvent{}, false
	}

	status, _ := state["status"].(string)
	input, _ := state["input"].(map[string]interface{})

	// Check for subagent (task tool)
	if tool == "task" && input != nil {
		subagentType, _ := input["subagent_type"].(string)
		description, _ := input["description"].(string)
		if subagentType != "" {
			return AgentEvent{
				Type:   "subagent",
				Tool:   tool,
				Status: status,
				Subagent: &AgentEventSubagent{
					AgentType:   subagentType,
					Description: description,
				},
			}, true
		}
	}

	var content string
	if input != nil {
		content = extractToolContent(tool, input)
	}

	return AgentEvent{
		Type:    "tool",
		Tool:    tool,
		Status:  status,
		Content: content,
	}, true
}

func parseOpenCodeTextEvent(part map[string]interface{}) (AgentEvent, bool) {
	if part == nil {
		return AgentEvent{}, false
	}
	text, _ := part["text"].(string)
	if text == "" {
		return AgentEvent{}, false
	}
	return AgentEvent{Type: "text", Content: text}, true
}

func parseOpenCodeReasoningEvent(part map[string]interface{}) (AgentEvent, bool) {
	if part == nil {
		return AgentEvent{}, false
	}
	text, _ := part["text"].(string)
	if text == "" {
		return AgentEvent{}, false
	}
	return AgentEvent{Type: "reasoning", Content: text}, true
}

func parseOpenCodeStepFinishEvent(part map[string]interface{}) (AgentEvent, bool) {
	if part == nil {
		return AgentEvent{}, false
	}

	cost, _ := part["cost"].(float64)
	tokens, _ := part["tokens"].(map[string]interface{})

	var ti AgentEventTokens
	if tokens != nil {
		if v, ok := tokens["input"].(float64); ok {
			ti.Input = int64(v)
		}
		if v, ok := tokens["output"].(float64); ok {
			ti.Output = int64(v)
		}
	}

	return AgentEvent{
		Type:   "step_finish",
		Cost:   cost,
		Tokens: ti,
	}, true
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

func extractToolContent(tool string, input map[string]interface{}) string {
	switch tool {
	case "Bash", "bash":
		if desc, ok := input["description"].(string); ok && desc != "" {
			return desc
		}
		if cmd, ok := input["command"].(string); ok {
			return cmd
		}
	case "Read", "read", "Write", "write", "Edit", "edit":
		if fp, ok := input["filePath"].(string); ok {
			return fp
		}
		if p, ok := input["path"].(string); ok {
			return p
		}
	case "Glob", "glob":
		if p, ok := input["pattern"].(string); ok {
			return p
		}
	case "Grep", "grep":
		if p, ok := input["pattern"].(string); ok {
			return p
		}
	case "Task", "task":
		if d, ok := input["description"].(string); ok {
			return d
		}
	}
	return ""
}

// GetOpenCodeModels fetches the list of available models from opencode.
func GetOpenCodeModels() ([]string, error) {
	cmd := exec.Command("opencode", "models")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get opencode models: %w", err)
	}

	var models []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "{") {
			models = append(models, line)
		}
	}

	return models, nil
}
