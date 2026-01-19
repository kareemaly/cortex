package ticket

import "testing"

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  string
	}{
		{
			name:  "simple title",
			title: "Add login",
			want:  "add-login",
		},
		{
			name:  "spaces become hyphens",
			title: "Add login functionality",
			want:  "add-login",
		},
		{
			name:  "underscores become hyphens",
			title: "add_login_func",
			want:  "add-login-func",
		},
		{
			name:  "removes special characters",
			title: "Fix bug #123!",
			want:  "fix-bug-123",
		},
		{
			name:  "collapses multiple hyphens",
			title: "fix---bug",
			want:  "fix-bug",
		},
		{
			name:  "trims hyphens from ends",
			title: "-fix bug-",
			want:  "fix-bug",
		},
		{
			name:  "truncates long title at word boundary",
			title: "This is a very long ticket title that exceeds the limit",
			want:  "this-is-a-very-long",
		},
		{
			name:  "empty title returns fallback",
			title: "",
			want:  "ticket",
		},
		{
			name:  "special chars only returns fallback",
			title: "!@#$%",
			want:  "ticket",
		},
		{
			name:  "mixed case becomes lowercase",
			title: "Fix BUG in Login",
			want:  "fix-bug-in-login",
		},
		{
			name:  "single long word truncates at max",
			title: "supercalifragilisticexpialidocious",
			want:  "supercalifragilistic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateSlug(tt.title)
			if got != tt.want {
				t.Errorf("GenerateSlug(%q) = %q, want %q", tt.title, got, tt.want)
			}
			if len(got) > maxSlugLength {
				t.Errorf("GenerateSlug(%q) = %q (len %d), exceeds max length %d", tt.title, got, len(got), maxSlugLength)
			}
		})
	}
}
