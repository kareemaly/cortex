package agent

import (
	"os"
	"path/filepath"
	"testing"
)

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
