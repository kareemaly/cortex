package spawn

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClaudeProjectSlug(t *testing.T) {
	tests := []struct {
		name string
		cwd  string
		want string
	}{
		{"empty", "", ""},
		{"absolute path", "/Users/foo/bar", "-Users-foo-bar"},
		{"trailing slash", "/Users/foo/bar/", "-Users-foo-bar"},
		{"root", "/", ""},
		{"single segment", "/tmp", "-tmp"},
		{"deeply nested", "/a/b/c/d/e", "-a-b-c-d-e"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := claudeProjectSlug(tc.cwd)
			if got != tc.want {
				t.Errorf("claudeProjectSlug(%q) = %q, want %q", tc.cwd, got, tc.want)
			}
		})
	}
}

func TestClaudeTranscriptPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}

	tests := []struct {
		name      string
		cwd       string
		sessionID string
		want      string
	}{
		{
			name:      "typical path",
			cwd:       "/Users/foo/bar",
			sessionID: "abc-123",
			want:      filepath.Join(home, ".claude", "projects", "-Users-foo-bar", "abc-123.jsonl"),
		},
		{
			name:      "trailing slash cwd",
			cwd:       "/Users/foo/bar/",
			sessionID: "abc-123",
			want:      filepath.Join(home, ".claude", "projects", "-Users-foo-bar", "abc-123.jsonl"),
		},
		{
			name:      "empty cwd",
			cwd:       "",
			sessionID: "abc",
			want:      "",
		},
		{
			name:      "empty sessionID",
			cwd:       "/Users/foo",
			sessionID: "",
			want:      "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ClaudeTranscriptPath(tc.cwd, tc.sessionID)
			if got != tc.want {
				t.Errorf("ClaudeTranscriptPath(%q, %q) = %q, want %q", tc.cwd, tc.sessionID, got, tc.want)
			}
		})
	}
}

func TestStartClaudeTailer_NoopOnEmptyInputs(t *testing.T) {
	// Smoke test: StartClaudeTailer must not panic or start a goroutine when
	// required inputs are empty. If it tried to run, it would dial daemonURL
	// and fail noisily; the no-op guard keeps it silent.
	StartClaudeTailer("", "/tmp/anything", "ticket-1", "/project", "http://127.0.0.1:4200")
	StartClaudeTailer("/tmp/transcript.jsonl", "", "ticket-1", "/project", "http://127.0.0.1:4200")
	StartClaudeTailer("/tmp/transcript.jsonl", "/tmp/liveness", "", "/project", "http://127.0.0.1:4200")
}
