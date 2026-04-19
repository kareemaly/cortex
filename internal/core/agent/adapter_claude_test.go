package agent

import (
	"os"
	"path/filepath"
	"testing"
)

// claudeFixture loads a golden pane snapshot from testdata. Shared with
// regression_claude_permission_test.go in the same package.
func claudeFixture(t *testing.T, name string) []byte {
	t.Helper()
	p := filepath.Join("testdata", "panes", "claude", name+".txt")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read fixture %s: %v", p, err)
	}
	return b
}

func TestClaudePhraseMatchesPermissionDialogs(t *testing.T) {
	a, ok := Get("claude")
	if !ok {
		t.Fatal("claude adapter not registered")
	}
	for _, name := range []string{"awaiting_input_bash", "awaiting_input_file_edit"} {
		t.Run(name, func(t *testing.T) {
			if phrase := a.MatchAwaitingInput(claudeFixture(t, name)); phrase == "" {
				t.Errorf("phrase did not match %s fixture", name)
			}
		})
	}
}

func TestClaudePhraseRejectsNonDialogFixtures(t *testing.T) {
	a, _ := Get("claude")
	for _, name := range []string{"working", "idle", "info_box_non_permission"} {
		t.Run(name, func(t *testing.T) {
			if phrase := a.MatchAwaitingInput(claudeFixture(t, name)); phrase != "" {
				t.Errorf("%s: unexpected match on phrase %q", name, phrase)
			}
		})
	}
}

func TestClaudeParseTranscriptLine(t *testing.T) {
	got := parseClaudeLine([]byte(`{"type":"user","content":"hi"}`))
	if !got.Activity {
		t.Error("non-empty line should report Activity=true")
	}
	if parseClaudeLine(nil).Activity {
		t.Error("empty line should yield Activity=false")
	}
}

func TestClaudeTranscriptPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	got := ClaudeTranscriptPath("/Users/foo/bar", "abc-123")
	want := filepath.Join(home, ".claude", "projects", "-Users-foo-bar", "abc-123.jsonl")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if ClaudeTranscriptPath("", "x") != "" {
		t.Error("empty cwd should return empty path")
	}
	if ClaudeTranscriptPath("/x", "") != "" {
		t.Error("empty session id should return empty path")
	}
}
