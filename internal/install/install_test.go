package install

import "testing"

func TestDefaultsDirForAgent(t *testing.T) {
	tests := []struct {
		agent string
		want  string
	}{
		{"claude", "claude-code"},
		{"", "claude-code"},
		{"opencode", "opencode"},
		{"copilot", "copilot"},
		{"unknown", "claude-code"},
	}

	for _, tt := range tests {
		t.Run(tt.agent, func(t *testing.T) {
			got := DefaultsDirForAgent(tt.agent)
			if got != tt.want {
				t.Errorf("DefaultsDirForAgent(%q) = %q, want %q", tt.agent, got, tt.want)
			}
		})
	}
}
