package detail

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestChangesTabLoadsOnActivationAndNavigatesCommits(t *testing.T) {
	loadCalls := 0
	model := New(
		"Ticket",
		"",
		[]Tab{
			{Label: "Overview", Content: "body", Kind: TabKindMarkdown},
			{Label: "Changes", Kind: TabKindChanges},
		},
		WithChangesLoader(func() tea.Msg {
			loadCalls++
			return ChangesLoaded(&ChangesData{
				Commits: []ChangeCommit{
					{
						SHA:        "aaaaaaaa",
						Subject:    "First commit",
						AuthorName: "Tester",
						AuthoredAt: time.Date(2026, 5, 9, 10, 0, 0, 0, time.UTC),
						Files:      []ChangeFile{{Path: "a.txt", Status: "modified", Additions: 1, Patch: "+one"}},
					},
					{
						SHA:        "bbbbbbbb",
						Subject:    "Second commit",
						AuthorName: "Tester",
						AuthoredAt: time.Date(2026, 5, 9, 11, 0, 0, 0, time.UTC),
						Files:      []ChangeFile{{Path: "b.txt", Status: "modified", Additions: 2, Patch: "+two"}},
					},
				},
			}, nil)
		}),
	)

	updated, cmd := model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	m := unwrapModel(t, updated)
	if cmd != nil {
		t.Fatalf("expected no load command before Changes tab is active")
	}

	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = unwrapModel(t, updated)
	if cmd == nil {
		t.Fatalf("expected load command when switching to Changes tab")
	}
	if !m.isChangesTabActive() {
		t.Fatalf("expected Changes tab to be active")
	}

	updated, _ = m.Update(cmd())
	m = unwrapModel(t, updated)
	if loadCalls != 1 {
		t.Fatalf("expected one lazy load call, got %d", loadCalls)
	}
	if len(m.changeCommits()) != 2 {
		t.Fatalf("expected 2 loaded commits, got %d", len(m.changeCommits()))
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = unwrapModel(t, updated)
	if m.selectedCommit != 1 {
		t.Fatalf("expected selected commit index 1, got %d", m.selectedCommit)
	}
}

func unwrapModel(t *testing.T, model tea.Model) Model {
	t.Helper()
	switch v := model.(type) {
	case Model:
		return v
	case *Model:
		return *v
	default:
		t.Fatalf("unexpected model type %T", model)
		return Model{}
	}
}

func TestColorizePatchStylesDiffLines(t *testing.T) {
	cases := []struct {
		line string
		want string
	}{
		{"diff --git a/file.txt b/file.txt", diffHeaderLineStyle.Render("diff --git a/file.txt b/file.txt")},
		{"index 1111111..2222222 100644", diffFileMetaStyle.Render("index 1111111..2222222 100644")},
		{"--- a/file.txt", diffFileMetaStyle.Render("--- a/file.txt")},
		{"+++ b/file.txt", diffFileMetaStyle.Render("+++ b/file.txt")},
		{"@@ -1 +1 @@", diffHunkStyle.Render("@@ -1 +1 @@")},
		{"-old line", diffDeletedStyle.Render("-old line")},
		{"+new line", diffAddedStyle.Render("+new line")},
		{" context", " context"},
	}

	for _, tc := range cases {
		if got := stylePatchLine(tc.line); got != tc.want {
			t.Fatalf("stylePatchLine(%q) = %q, want %q", tc.line, got, tc.want)
		}
	}

	patch := strings.Join([]string{
		"diff --git a/file.txt b/file.txt",
		"index 1111111..2222222 100644",
		"--- a/file.txt",
		"+++ b/file.txt",
		"@@ -1 +1 @@",
		"-old line",
		"+new line",
		" context",
	}, "\n")

	rendered := colorizePatch(patch)
	for _, needle := range []string{
		"diff --git a/file.txt b/file.txt",
		"index 1111111..2222222 100644",
		"@@ -1 +1 @@",
		"-old line",
		"+new line",
	} {
		if !strings.Contains(rendered, needle) {
			t.Fatalf("expected rendered patch to contain %q", needle)
		}
	}
}
