package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kareemaly/cortex/internal/session"
)

func codexFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "panes", "codex", name+".txt"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return b
}

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

func TestCodexPhraseMatchesApproval(t *testing.T) {
	a, _ := Get("codex")
	if phrase := a.MatchAwaitingInput(codexFixture(t, "awaiting_input_command")); phrase == "" {
		t.Error("expected phrase match on approval fixture")
	}
}

func TestCodexPhraseRejectsNonApproval(t *testing.T) {
	a, _ := Get("codex")
	for _, name := range []string{"working", "idle", "info_banner_non_approval"} {
		t.Run(name, func(t *testing.T) {
			if phrase := a.MatchAwaitingInput(codexFixture(t, name)); phrase != "" {
				t.Errorf("%s: unexpected phrase match %q", name, phrase)
			}
		})
	}
}

func TestFindCodexRollout(t *testing.T) {
	dir := t.TempDir()
	deep := filepath.Join(dir, "sessions", "2026", "04", "18")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(deep, "rollout-1234-abc.jsonl")
	if err := os.WriteFile(target, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := findCodexRollout(dir)
	if got != target {
		t.Errorf("got %q, want %q", got, target)
	}
}

func TestFindCodexRolloutAbsent(t *testing.T) {
	dir := t.TempDir()
	if got := findCodexRollout(dir); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
	if got := findCodexRollout(""); got != "" {
		t.Errorf("empty arg should return empty, got %q", got)
	}
}
