package spawn

import (
	"encoding/json"
	"fmt"
)

// OpenCodeConfigContent represents the top-level OpenCode config JSON structure.
// This is passed via the OPENCODE_CONFIG_CONTENT environment variable.
type OpenCodeConfigContent struct {
	Agent map[string]OpenCodeAgentConfig `json:"agent"`
	MCP   map[string]OpenCodeMCPConfig   `json:"mcp"`
}

// OpenCodeAgentConfig represents an agent definition in OpenCode's config.
type OpenCodeAgentConfig struct {
	Description string            `json:"description"`
	Mode        string            `json:"mode"`
	Prompt      string            `json:"prompt"`
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
func GenerateOpenCodeConfigContent(claudeConfig *ClaudeMCPConfig, systemPrompt string) (string, error) {
	config := OpenCodeConfigContent{
		Agent: map[string]OpenCodeAgentConfig{
			"cortex": {
				Description: "Cortex ticket agent",
				Mode:        "bypassPermissions",
				Prompt:      systemPrompt,
				Permission:  map[string]string{"*": "allow"},
			},
		},
		MCP: make(map[string]OpenCodeMCPConfig),
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
