package install

import (
	"os/exec"

	"github.com/kareemaly/cortex/internal/daemon/config"
)

// AgentAvailability holds the results of checking for available agents.
type AgentAvailability struct {
	ClaudeAvailable   bool
	ClaudePath        string
	OpenCodeAvailable bool
	OpenCodePath      string
}

// DetectAgents checks which AI agents are available on the system.
func DetectAgents() AgentAvailability {
	var a AgentAvailability
	if path, err := exec.LookPath("claude"); err == nil {
		a.ClaudeAvailable = true
		a.ClaudePath = path
	}
	if path, err := exec.LookPath("opencode"); err == nil {
		a.OpenCodeAvailable = true
		a.OpenCodePath = path
	}
	return a
}

// OnlyAgent returns the agent name if exactly one is available, or "" if 0 or 2.
func (a AgentAvailability) OnlyAgent() string {
	if a.ClaudeAvailable && !a.OpenCodeAvailable {
		return "claude"
	}
	if a.OpenCodeAvailable && !a.ClaudeAvailable {
		return "opencode"
	}
	return ""
}

// AgentCount returns the number of available agents (0, 1, or 2).
func (a AgentAvailability) AgentCount() int {
	count := 0
	if a.ClaudeAvailable {
		count++
	}
	if a.OpenCodeAvailable {
		count++
	}
	return count
}

// DefaultClaudeVariants returns the standard set of Claude agent variants.
func DefaultClaudeVariants() map[string]config.AgentVariant {
	return map[string]config.AgentVariant{
		"claude-opus": {
			Agent: "claude",
			Args:  []string{"--dangerously-skip-permissions"},
		},
		"claude-opus-plan": {
			Agent: "claude",
			Args:  []string{"--allow-dangerously-skip-permissions", "--permission-mode", "plan"},
		},
		"claude-sonnet": {
			Agent: "claude",
			Args:  []string{"--dangerously-skip-permissions", "--model", "claude-sonnet-4-6"},
		},
		"claude-sonnet-plan": {
			Agent: "claude",
			Args:  []string{"--allow-dangerously-skip-permissions", "--permission-mode", "plan", "--model", "claude-sonnet-4-6"},
		},
	}
}

// DefaultOpenCodeVariants returns the standard set of OpenCode agent variants.
func DefaultOpenCodeVariants() map[string]config.AgentVariant {
	return map[string]config.AgentVariant{
		"opencode": {
			Agent: "opencode",
			Args:  []string{},
		},
		"opencode-plan": {
			Agent: "opencode",
			Args:  []string{"--agent", "plan"},
		},
	}
}
