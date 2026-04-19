package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kareemaly/cortex/internal/session"
)

func opencodeFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "panes", "opencode", name+".txt"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return b
}

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

func TestOpenCodePhraseMatchesPermissionFallback(t *testing.T) {
	a, _ := Get("opencode")
	if phrase := a.MatchAwaitingInput(opencodeFixture(t, "awaiting_input_permission")); phrase == "" {
		t.Error("expected phrase match on permission fixture")
	}
}

func TestOpenCodePhraseRejectsNonPermission(t *testing.T) {
	a, _ := Get("opencode")
	for _, name := range []string{"working", "info_divider_non_permission"} {
		t.Run(name, func(t *testing.T) {
			if phrase := a.MatchAwaitingInput(opencodeFixture(t, name)); phrase != "" {
				t.Errorf("%s: unexpected phrase match %q", name, phrase)
			}
		})
	}
}
