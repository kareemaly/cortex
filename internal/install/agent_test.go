package install

import "testing"

func TestDetectAgentFromExtend(t *testing.T) {
	tests := []struct {
		name   string
		extend string
		want   string
	}{
		{"claude legacy path", "~/.cortex/defaults/claude-code", "claude"},
		{"opencode legacy path", "~/.cortex/defaults/opencode", "opencode"},
		{"already migrated", "~/.cortex/defaults/main", ""},
		{"empty string", "", ""},
		{"unrecognized path", "/some/other/path", ""},
		{"claude in full path", "/home/user/.cortex/defaults/claude-code", "claude"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectAgentFromExtend(tt.extend)
			if got != tt.want {
				t.Errorf("DetectAgentFromExtend(%q) = %q, want %q", tt.extend, got, tt.want)
			}
		})
	}
}

func TestAgentAvailability_OnlyAgent(t *testing.T) {
	tests := []struct {
		name string
		a    AgentAvailability
		want string
	}{
		{"claude only", AgentAvailability{ClaudeAvailable: true}, "claude"},
		{"opencode only", AgentAvailability{OpenCodeAvailable: true}, "opencode"},
		{"both available", AgentAvailability{ClaudeAvailable: true, OpenCodeAvailable: true}, ""},
		{"neither available", AgentAvailability{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.OnlyAgent()
			if got != tt.want {
				t.Errorf("OnlyAgent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAgentAvailability_AgentCount(t *testing.T) {
	tests := []struct {
		name string
		a    AgentAvailability
		want int
	}{
		{"none", AgentAvailability{}, 0},
		{"claude only", AgentAvailability{ClaudeAvailable: true}, 1},
		{"opencode only", AgentAvailability{OpenCodeAvailable: true}, 1},
		{"both", AgentAvailability{ClaudeAvailable: true, OpenCodeAvailable: true}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.AgentCount()
			if got != tt.want {
				t.Errorf("AgentCount() = %d, want %d", got, tt.want)
			}
		})
	}
}
