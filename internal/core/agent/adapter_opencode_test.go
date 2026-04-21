package agent

import (
	"testing"

	"github.com/kareemaly/cortex/internal/session"
)

func TestOpenCodeParseTranscriptLineCanonicalStatuses(t *testing.T) {
	// The plugin emits only canonical names; ParseTranscriptLine forwards them verbatim.
	cases := []struct {
		in   string
		want session.AgentStatus
	}{
		{`{"status":"working"}`, session.AgentStatusWorking},
		{`{"status":"awaiting_input"}`, session.AgentStatusAwaitingInput},
		{`{"status":"idle"}`, session.AgentStatusIdle},
		{`{"status":"error"}`, session.AgentStatusError},
	}
	for _, tc := range cases {
		got := parseOpenCodeLine([]byte(tc.in)).Status
		if got != tc.want {
			t.Errorf("parseOpenCodeLine(%q).Status = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestOpenCodeParseTranscriptLineCarriesToolAndWork(t *testing.T) {
	got := parseOpenCodeLine([]byte(`{"status":"working","tool":"Bash","work":"make test"}`))
	if got.Tool != "Bash" {
		t.Errorf("tool = %q, want Bash", got.Tool)
	}
	if got.Work != "make test" {
		t.Errorf("work = %q, want 'make test'", got.Work)
	}
}
