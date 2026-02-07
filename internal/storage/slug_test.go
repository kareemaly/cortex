package storage

import "testing"

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		fallback string
		want     string
	}{
		{name: "simple title", title: "Add login", fallback: "ticket", want: "add-login"},
		{name: "spaces become hyphens", title: "Add login functionality", fallback: "ticket", want: "add-login"},
		{name: "underscores become hyphens", title: "add_login_func", fallback: "ticket", want: "add-login-func"},
		{name: "removes special characters", title: "Fix bug #123!", fallback: "ticket", want: "fix-bug-123"},
		{name: "collapses multiple hyphens", title: "fix---bug", fallback: "ticket", want: "fix-bug"},
		{name: "trims hyphens from ends", title: "-fix bug-", fallback: "ticket", want: "fix-bug"},
		{name: "truncates long title at word boundary", title: "This is a very long ticket title that exceeds the limit", fallback: "ticket", want: "this-is-a-very-long"},
		{name: "empty title returns ticket fallback", title: "", fallback: "ticket", want: "ticket"},
		{name: "empty title returns doc fallback", title: "", fallback: "doc", want: "doc"},
		{name: "special chars only returns fallback", title: "!@#$%", fallback: "ticket", want: "ticket"},
		{name: "mixed case becomes lowercase", title: "Fix BUG in Login", fallback: "ticket", want: "fix-bug-in-login"},
		{name: "single long word truncates at max", title: "supercalifragilisticexpialidocious", fallback: "ticket", want: "supercalifragilistic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateSlug(tt.title, tt.fallback)
			if got != tt.want {
				t.Errorf("GenerateSlug(%q, %q) = %q, want %q", tt.title, tt.fallback, got, tt.want)
			}
			if len(got) > maxSlugLength {
				t.Errorf("GenerateSlug(%q, %q) = %q (len %d), exceeds max length %d", tt.title, tt.fallback, got, len(got), maxSlugLength)
			}
		})
	}
}

func TestShortID(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"a1b2c3d4-e5f6-7890-abcd-ef0123456789", "a1b2c3d4"},
		{"abcdefgh", "abcdefgh"},
		{"short", "short"},
		{"", ""},
	}

	for _, tt := range tests {
		got := ShortID(tt.id)
		if got != tt.want {
			t.Errorf("ShortID(%q) = %q, want %q", tt.id, got, tt.want)
		}
	}
}

func TestDirName(t *testing.T) {
	got := DirName("Fix Auth Bug", "a1b2c3d4-e5f6-7890", "ticket")
	want := "fix-auth-bug-a1b2c3d4"
	if got != want {
		t.Errorf("DirName() = %q, want %q", got, want)
	}

	got = DirName("", "a1b2c3d4-e5f6-7890", "doc")
	want = "doc-a1b2c3d4"
	if got != want {
		t.Errorf("DirName() = %q, want %q", got, want)
	}
}
