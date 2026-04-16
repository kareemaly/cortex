package spawn

import "testing"

func TestParseOpenCodeLine(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		wantStatus string
		wantTool   string
	}{
		{
			name:       "status only",
			line:       `{"status":"in_progress"}`,
			wantStatus: "in_progress",
		},
		{
			name:       "status with tool",
			line:       `{"status":"in_progress","tool":"bash"}`,
			wantStatus: "in_progress",
			wantTool:   "bash",
		},
		{
			name:       "idle",
			line:       `{"status":"idle"}`,
			wantStatus: "idle",
		},
		{
			name:       "waiting_permission",
			line:       `{"status":"waiting_permission"}`,
			wantStatus: "waiting_permission",
		},
		{
			name: "malformed json ignored",
			line: `not json`,
		},
		{
			name: "empty object ignored",
			line: `{}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseOpenCodeLine([]byte(tc.line))
			if got.Status != tc.wantStatus {
				t.Errorf("Status = %q, want %q", got.Status, tc.wantStatus)
			}
			if got.Tool != tc.wantTool {
				t.Errorf("Tool = %q, want %q", got.Tool, tc.wantTool)
			}
		})
	}
}

func TestOpenCodeStatusFilePath(t *testing.T) {
	got := OpenCodeStatusFilePath("ticket-1", "/tmp/cortex")
	want := "/tmp/cortex/cortex-opencode-status-ticket-1.jsonl"
	if got != want {
		t.Errorf("OpenCodeStatusFilePath = %q, want %q", got, want)
	}

	// Empty configDir falls back to os.TempDir().
	got = OpenCodeStatusFilePath("ticket-1", "")
	if got == "" {
		t.Error("expected non-empty path when configDir is empty")
	}
}
