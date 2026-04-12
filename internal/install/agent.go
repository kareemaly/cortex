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
	CodexAvailable    bool
	CodexPath         string
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
	if path, err := exec.LookPath("codex"); err == nil {
		a.CodexAvailable = true
		a.CodexPath = path
	}
	return a
}

// OnlyAgent returns the agent name if exactly one is available, or "" if 0 or more than 1.
func (a AgentAvailability) OnlyAgent() string {
	if a.AgentCount() != 1 {
		return ""
	}
	if a.ClaudeAvailable {
		return "claude"
	}
	if a.OpenCodeAvailable {
		return "opencode"
	}
	if a.CodexAvailable {
		return "codex"
	}
	return ""
}

// AgentCount returns the number of available agents.
func (a AgentAvailability) AgentCount() int {
	count := 0
	if a.ClaudeAvailable {
		count++
	}
	if a.OpenCodeAvailable {
		count++
	}
	if a.CodexAvailable {
		count++
	}
	return count
}

// DefaultClaudeVariants returns the standard set of Claude agent variants.
func DefaultClaudeVariants() map[string]config.AgentVariant {
	return map[string]config.AgentVariant{
		"claude-opus": {
			Agent: "claude",
			Args:  []string{"--permission-mode", "auto"},
		},
		"claude-opus-plan": {
			Agent: "claude",
			Args:  []string{"--permission-mode", "plan"},
		},
		"claude-sonnet": {
			Agent: "claude",
			Args:  []string{"--permission-mode", "auto", "--model", "claude-sonnet-4-6"},
		},
		"claude-sonnet-plan": {
			Agent: "claude",
			Args:  []string{"--permission-mode", "plan", "--model", "claude-sonnet-4-6"},
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

// DefaultCodexVariants returns the standard set of Codex agent variants.
func DefaultCodexVariants() map[string]config.AgentVariant {
	return map[string]config.AgentVariant{
		"codex": {
			Agent: "codex",
			Args:  []string{"--full-auto", "--model", "gpt-5.4"},
		},
		"codex-plan": {
			Agent: "codex",
			Args:  []string{"--sandbox", "read-only", "--ask-for-approval", "never", "--model", "gpt-5.4"},
		},
	}
}
