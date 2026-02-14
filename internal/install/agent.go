package install

import "os/exec"

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
