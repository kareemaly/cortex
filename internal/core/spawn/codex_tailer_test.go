package spawn

import (
	"strings"
	"testing"
)

func TestParseRolloutLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{
			name:     "session_meta maps to idle",
			line:     `{"timestamp":"2026-04-08T10:22:26Z","type":"session_meta","payload":{"id":"abc","originator":"codex_exec","cli_version":"0.1.0"}}`,
			expected: "idle",
		},
		{
			name:     "event_msg task_started maps to in_progress",
			line:     `{"timestamp":"2026-04-08T10:22:30Z","type":"event_msg","payload":{"type":"task_started","turn_id":"01234","model_context_window":128000,"collaboration_mode_kind":"default"}}`,
			expected: "in_progress",
		},
		{
			name:     "event_msg task_complete maps to idle",
			line:     `{"timestamp":"2026-04-08T10:22:45Z","type":"event_msg","payload":{"type":"task_complete","turn_id":"01234","last_agent_message":"Done."}}`,
			expected: "idle",
		},
		{
			name:     "event_msg token_count is ignored",
			line:     `{"timestamp":"2026-04-08T10:22:31Z","type":"event_msg","payload":{"type":"token_count","info":{"input_tokens":100}}}`,
			expected: "",
		},
		{
			name:     "event_msg agent_message is ignored",
			line:     `{"timestamp":"2026-04-08T10:22:40Z","type":"event_msg","payload":{"type":"agent_message","message":"hello","phase":"commentary"}}`,
			expected: "",
		},
		{
			name:     "event_msg user_message is ignored",
			line:     `{"timestamp":"2026-04-08T10:22:28Z","type":"event_msg","payload":{"type":"user_message","message":"do something"}}`,
			expected: "",
		},
		{
			name:     "response_item message is ignored",
			line:     `{"timestamp":"2026-04-08T10:22:29Z","type":"response_item","payload":{"type":"message","role":"user"}}`,
			expected: "",
		},
		{
			name:     "response_item function_call is ignored",
			line:     `{"timestamp":"2026-04-08T10:22:32Z","type":"response_item","payload":{"type":"function_call","name":"shell","call_id":"c1"}}`,
			expected: "",
		},
		{
			name:     "turn_context is ignored",
			line:     `{"timestamp":"2026-04-08T10:22:27Z","type":"turn_context","payload":{"model":"gpt-5.4"}}`,
			expected: "",
		},
		{
			name:     "malformed json returns empty",
			line:     `not valid json`,
			expected: "",
		},
		{
			name:     "empty line returns empty",
			line:     ``,
			expected: "",
		},
		{
			name: "large session_meta with base_instructions is handled",
			// Simulate a large payload by embedding a long base_instructions field.
			line:     `{"timestamp":"2026-04-08T10:22:26Z","type":"session_meta","payload":{"id":"xyz","base_instructions":"` + strings.Repeat("x", 70000) + `"}}`,
			expected: "idle",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseRolloutLine([]byte(tc.line))
			if got.Status != tc.expected {
				preview := tc.line
				if len(preview) > 80 {
					preview = preview[:80]
				}
				t.Errorf("parseRolloutLine(%q).Status = %q, want %q", preview, got.Status, tc.expected)
			}
		})
	}
}
