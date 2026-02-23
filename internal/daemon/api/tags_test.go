package api

import (
	"net/http"
	"testing"
)

func TestTags_EmptyStores(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	resp := ts.makeRequest(t, http.MethodGet, "/tags", nil)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusOK)

	result := decode[ListTagsResponse](t, resp)
	if len(result.Tags) != 0 {
		t.Errorf("expected 0 tags, got %d", len(result.Tags))
	}
}

func TestTags_TicketTagsOnly(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	_, _ = ts.store.Create("Ticket 1", "body", "", nil, nil, []string{"api", "bug"}, "")
	_, _ = ts.store.Create("Ticket 2", "body", "", nil, nil, []string{"API", "feature"}, "")
	_, _ = ts.store.Create("Ticket 3", "body", "", nil, nil, []string{"bug"}, "")

	resp := ts.makeRequest(t, http.MethodGet, "/tags", nil)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusOK)

	result := decode[ListTagsResponse](t, resp)
	if len(result.Tags) != 3 {
		t.Fatalf("expected 3 tags, got %d: %+v", len(result.Tags), result.Tags)
	}

	// Sorted by count descending: api(2), bug(2), feature(1)
	// api and bug tie at 2, alphabetical tiebreak: api < bug
	if result.Tags[0].Name != "api" || result.Tags[0].Count != 2 {
		t.Errorf("expected first tag api:2, got %s:%d", result.Tags[0].Name, result.Tags[0].Count)
	}
	if result.Tags[1].Name != "bug" || result.Tags[1].Count != 2 {
		t.Errorf("expected second tag bug:2, got %s:%d", result.Tags[1].Name, result.Tags[1].Count)
	}
	if result.Tags[2].Name != "feature" || result.Tags[2].Count != 1 {
		t.Errorf("expected third tag feature:1, got %s:%d", result.Tags[2].Name, result.Tags[2].Count)
	}
}

func TestTags_TicketTagsOnlyNoConclusionManager(t *testing.T) {
	// Uses the standard setupUnitServer which doesn't set ConclusionStoreManager
	ts := setupUnitServer(t)
	defer ts.Close()

	_, _ = ts.store.Create("Ticket 1", "body", "", nil, nil, []string{"api"}, "")

	resp := ts.makeRequest(t, http.MethodGet, "/tags", nil)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusOK)

	result := decode[ListTagsResponse](t, resp)
	if len(result.Tags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(result.Tags))
	}
	if result.Tags[0].Name != "api" {
		t.Errorf("expected tag api, got %q", result.Tags[0].Name)
	}
}
