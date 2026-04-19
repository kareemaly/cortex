package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kareemaly/cortex/internal/session"
)

// fixturesDir returns the absolute path to the claude pane fixtures for
// a given scenario name (minus the .txt extension).
func claudeFixture(t *testing.T, name string) []byte {
	t.Helper()
	p := filepath.Join("testdata", "panes", "claude", name+".txt")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read fixture %s: %v", p, err)
	}
	return b
}

func TestClaudePaneMatchesPermissionDialogs(t *testing.T) {
	a, ok := Get("claude")
	if !ok {
		t.Fatal("claude adapter not registered")
	}
	cases := []string{"awaiting_input_bash", "awaiting_input_file_edit"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			raw := claudeFixture(t, name)
			_, implied, ok := a.PanePatterns.MatchFirst(raw)
			if !ok {
				t.Fatalf("expected match on %s fixture", name)
			}
			if implied != session.AgentStatusAwaitingInput {
				t.Errorf("implied = %v, want awaiting_input", implied)
			}
		})
	}
}

func TestClaudePaneRejectsWorkingFixture(t *testing.T) {
	a, _ := Get("claude")
	raw := claudeFixture(t, "working")
	if _, _, ok := a.PanePatterns.MatchFirst(raw); ok {
		t.Error("working fixture must not match any permission pattern")
	}
}

func TestClaudePaneRejectsInfoBoxFixture(t *testing.T) {
	// Negative case: a non-permission info card has the Claude border but
	// lacks the ❯ focus caret and Yes/No options. The border-AND-anchor
	// discipline must reject it.
	a, _ := Get("claude")
	raw := claudeFixture(t, "info_box_non_permission")
	if _, _, ok := a.PanePatterns.MatchFirst(raw); ok {
		t.Error("non-permission info box must not match")
	}
}

func TestClaudePaneIdleFixture(t *testing.T) {
	a, _ := Get("claude")
	raw := claudeFixture(t, "idle")
	if _, _, ok := a.PanePatterns.MatchFirst(raw); ok {
		t.Error("idle fixture must not match")
	}
}

func TestClaudeParseLine(t *testing.T) {
	got := parseClaudeLine([]byte(`{"type":"user","content":"hi"}`))
	if got.Status != session.AgentStatusWorking {
		t.Errorf("got = %v, want working", got.Status)
	}
	// Empty line is ignored.
	if parseClaudeLine(nil).Status != "" {
		t.Error("empty line should yield empty status")
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
