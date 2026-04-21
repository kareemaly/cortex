package agent

import (
	"testing"

	"github.com/kareemaly/cortex/internal/session"
)

func TestCodexParseTranscriptLine(t *testing.T) {
	cases := []struct {
		name string
		line string
		want session.AgentStatus
	}{
		{"session_meta", `{"type":"session_meta","payload":{}}`, session.AgentStatusIdle},
		{"task_started", `{"type":"event_msg","payload":{"type":"task_started"}}`, session.AgentStatusWorking},
		{"task_complete", `{"type":"event_msg","payload":{"type":"task_complete"}}`, session.AgentStatusIdle},
		{"ignored_type", `{"type":"event_msg","payload":{"type":"unrelated"}}`, ""},
		{"garbage", `not json`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseCodexLine([]byte(tc.line)).Status
			if got != tc.want {
				t.Errorf("parseCodexLine(%q).Status = %q, want %q", tc.line, got, tc.want)
			}
		})
	}
}
