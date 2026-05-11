package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/kareemaly/cortex/internal/architectsession"
	"github.com/kareemaly/cortex/internal/ticket"
)

func TestGetConclusion_ExposesCommitsAndRejection(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	projectPath := ts.projectRoot

	created, _ := ts.store.Create("test-ticket", "body", nil, nil, "")
	meta := &ticket.TicketConclusionMeta{
		StartedAt:       time.Now().UTC().Add(-2 * time.Minute),
		ConcludedAt:     time.Now().UTC(),
		Agent:           "codex",
		Commits:         []string{"abc123", "def456"},
		Rejected:        true,
		RejectionReason: "no shippable change",
	}

	if err := ts.store.WriteConclusion(created.ID, meta, "done report"); err != nil {
		t.Fatal(err)
	}

	_ = architectsession.EnsureDir(projectPath)

	resp := ts.makeRequest(t, http.MethodGet, "/conclusions/"+created.ID, nil)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusOK)

	result := decode[ConclusionResponse](t, resp)
	if len(result.Commits) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(result.Commits))
	}
	if !result.Rejected {
		t.Fatal("expected rejected conclusion")
	}
	if result.RejectionReason != "no shippable change" {
		t.Fatalf("expected rejection reason, got %q", result.RejectionReason)
	}
}
