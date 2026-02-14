package spawn

import (
	"encoding/json"
	"fmt"
)

// OpenCodeConfigContent represents the top-level OpenCode config JSON structure.
// This is passed via the OPENCODE_CONFIG_CONTENT environment variable.
type OpenCodeConfigContent struct {
	Agent        map[string]OpenCodeAgentConfig `json:"agent"`
	MCP          map[string]OpenCodeMCPConfig   `json:"mcp"`
	Instructions []string                       `json:"instructions,omitempty"`
}

// OpenCodeAgentConfig represents an agent definition in OpenCode's config.
type OpenCodeAgentConfig struct {
	Description string            `json:"description"`
	Mode        string            `json:"mode"`
	Prompt      string            `json:"prompt,omitempty"`
	Permission  map[string]string `json:"permission"`
}

// OpenCodeMCPConfig represents an MCP server definition in OpenCode's config.
type OpenCodeMCPConfig struct {
	Type        string            `json:"type"`
	Command     []string          `json:"command"`
	Environment map[string]string `json:"environment,omitempty"`
}

// GenerateOpenCodeConfigContent transforms a ClaudeMCPConfig and system prompt
// into the JSON string expected by OpenCode's OPENCODE_CONFIG_CONTENT env var.
// For ticket agents, the system prompt is provided via instructions (file path)
// instead of embedding in agent.prompt, so OpenCode's built-in provider prompt is preserved.
func GenerateOpenCodeConfigContent(claudeConfig *ClaudeMCPConfig, systemPrompt string, agentType AgentType, systemPromptFilePath string) (string, error) {
	config := OpenCodeConfigContent{
		Agent: map[string]OpenCodeAgentConfig{
			"cortex": {
				Description: "Cortex ticket agent",
				Mode:        "bypassPermissions",
				Permission:  map[string]string{"*": "allow"},
			},
		},
		MCP: make(map[string]OpenCodeMCPConfig),
	}

	if agentType == AgentTypeTicketAgent {
		// Ticket agents use instructions (file paths appended to system prompt)
		// so OpenCode's built-in provider prompt is preserved.
		if systemPromptFilePath != "" {
			config.Instructions = []string{systemPromptFilePath}
		}
	} else {
		// Architect/meta agents replace the system prompt entirely.
		config.Agent["cortex"] = OpenCodeAgentConfig{
			Description: "Cortex ticket agent",
			Mode:        "bypassPermissions",
			Prompt:      systemPrompt,
			Permission:  map[string]string{"*": "allow"},
		}
	}

	// Transform each MCP server from Claude format to OpenCode format
	for name, server := range claudeConfig.MCPServers {
		// Combine Command + Args into a single command array
		command := make([]string, 0, 1+len(server.Args))
		command = append(command, server.Command)
		command = append(command, server.Args...)

		config.MCP[name] = OpenCodeMCPConfig{
			Type:        "local",
			Command:     command,
			Environment: server.Env,
		}
	}

	data, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("marshal opencode config: %w", err)
	}

	return string(data), nil
}
