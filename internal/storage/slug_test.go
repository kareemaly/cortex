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

func TestSanitizeTmuxName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "simple name", input: "myproject", want: "myproject"},
		{name: "spaces become hyphens", input: "Footprint Management", want: "Footprint-Management"},
		{name: "multiple spaces", input: "My  Project  Name", want: "My--Project--Name"},
		{name: "special chars replaced", input: "my@project#name", want: "my-project-name"},
		{name: "starts with hyphen becomes underscore", input: "-myproject", want: "_myproject"},
		{name: "starts with space becomes underscore", input: " myproject", want: "_myproject"},
		{name: "keeps underscores", input: "my_project_name", want: "my_project_name"},
		{name: "keeps hyphens", input: "my-project-name", want: "my-project-name"},
		{name: "mixed case preserved", input: "MyProject", want: "MyProject"},
		{name: "numbers preserved", input: "project123", want: "project123"},
		{name: "colons replaced", input: "project:name", want: "project-name"},
		{name: "periods replaced", input: "project.name", want: "project-name"},
		{name: "empty string", input: "", want: ""},
		{name: "only special chars", input: "@#$%", want: "_---"},
		{name: "unicode replaced", input: "café-project", want: "caf--project"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeTmuxName(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeTmuxName(%q) = %q, want %q", tt.input, got, tt.want)
			}
			if len(got) > maxTmuxNameLength {
				t.Errorf("SanitizeTmuxName(%q) = %q (len %d), exceeds max length %d", tt.input, got, len(got), maxTmuxNameLength)
			}
		})
	}
}

func TestSanitizeTmuxName_Length(t *testing.T) {
	longName := ""
	for i := 0; i < 200; i++ {
		longName += "a"
	}

	got := SanitizeTmuxName(longName)
	if len(got) != maxTmuxNameLength {
		t.Errorf("SanitizeTmuxName() length = %d, want %d", len(got), maxTmuxNameLength)
	}
}

func TestSanitizeTmuxName_Consistency(t *testing.T) {
	input := "Footprint Management"

	first := SanitizeTmuxName(input)
	second := SanitizeTmuxName(input)

	if first != second {
		t.Errorf("SanitizeTmuxName is not consistent: first=%q, second=%q", first, second)
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
