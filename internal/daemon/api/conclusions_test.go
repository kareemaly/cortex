package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/kareemaly/cortex/internal/conclusion"
)

func TestGetConclusion_ExposesCommitsAndRejection(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	created, err := ts.conclusionStore.Create(conclusion.CreateParams{
		Type:            "work",
		TicketID:        "ticket-1",
		Repo:            "/repo",
		Body:            "done report",
		StartedAt:       time.Now().UTC().Add(-2 * time.Minute),
		Commits:         []string{"abc123", "def456"},
		Rejected:        true,
		RejectionReason: "no shippable change",
	})
	if err != nil {
		t.Fatal(err)
	}

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
