package commands

import (
	"testing"
	"time"

	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/cli/tui/detail"
)

func TestBuildTicketTabsIncludesChangesOnlyWhenConclusionHasCommits(t *testing.T) {
	ticket := &sdk.TicketResponse{ID: "t1", Title: "Ticket"}

	tabs := buildTicketTabs(ticket, nil, &sdk.ConclusionResponse{ID: "c1"}, "")
	if hasTabKind(tabs, detail.TabKindChanges) {
		t.Fatalf("expected no Changes tab without commits")
	}

	tabs = buildTicketTabs(ticket, nil, &sdk.ConclusionResponse{ID: "c1", Commits: []string{"abc123"}}, "")
	if !hasTabKind(tabs, detail.TabKindChanges) {
		t.Fatalf("expected Changes tab when conclusion has commits")
	}
}

func TestBuildChangesDataMapsResponse(t *testing.T) {
	oldPath := "old.txt"
	authoredAt := time.Date(2026, 5, 9, 12, 34, 56, 0, time.UTC)
	resp := &sdk.DiffsResponse{
		Repo: "/tmp/repo",
		Commits: []sdk.CommitDiffResponse{
			{
				SHA:        "abcdef123456",
				Subject:    "Update file",
				AuthorName: "Test User",
				AuthoredAt: authoredAt,
				Files: []sdk.DiffFileResponse{
					{
						Path:      "new.txt",
						OldPath:   &oldPath,
						Status:    "renamed",
						IsBinary:  false,
						Additions: 3,
						Deletions: 1,
						Patch:     "diff --git a/old.txt b/new.txt",
					},
				},
			},
		},
	}

	changes := buildChangesData(resp)
	if changes == nil {
		t.Fatalf("expected mapped changes data")
	}
	if changes.Repo != resp.Repo {
		t.Fatalf("expected repo %q, got %q", resp.Repo, changes.Repo)
	}
	if len(changes.Commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(changes.Commits))
	}
	if changes.Commits[0].Files[0].OldPath != oldPath {
		t.Fatalf("expected old path %q, got %q", oldPath, changes.Commits[0].Files[0].OldPath)
	}
}

func hasTabKind(tabs []detail.Tab, kind detail.TabKind) bool {
	for _, tab := range tabs {
		if tab.Kind == kind {
			return true
		}
	}
	return false
}
