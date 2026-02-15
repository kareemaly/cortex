package api

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/kareemaly/cortex/internal/docs"
	"github.com/kareemaly/cortex/internal/events"
	"github.com/kareemaly/cortex/internal/ticket"
)

// setupUnitServerWithDocs creates a test server with both ticket and docs stores.
func setupUnitServerWithDocs(t *testing.T) (*unitServer, *docs.Store) {
	t.Helper()

	tmpDir := t.TempDir()

	cortexDir := filepath.Join(tmpDir, ".cortex")
	if err := os.MkdirAll(cortexDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cortexDir, "cortex.yaml"), []byte("name: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	ticketsDir := filepath.Join(tmpDir, "tickets")
	store, err := ticket.NewStore(ticketsDir, nil, "")
	if err != nil {
		t.Fatalf("failed to create ticket store: %v", err)
	}

	docsDir := filepath.Join(tmpDir, "docs")
	docsStore, err := docs.NewStore(docsDir, nil, "")
	if err != nil {
		t.Fatalf("failed to create docs store: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	storeManager := NewStoreManager(logger, nil)
	storeManager.stores[tmpDir] = store

	docsStoreManager := NewDocsStoreManager(logger, nil)
	docsStoreManager.stores[tmpDir] = docsStore

	sessionManager := NewSessionManager(logger)

	deps := &Dependencies{
		StoreManager:     storeManager,
		DocsStoreManager: docsStoreManager,
		SessionManager:   sessionManager,
		TmuxManager:      nil,
		Bus:              events.NewBus(),
		Logger:           logger,
	}

	return &unitServer{
		Server:      httptest.NewServer(NewRouter(deps, deps.Logger)),
		store:       store,
		projectRoot: tmpDir,
	}, docsStore
}

func TestTags_EmptyStores(t *testing.T) {
	ts, _ := setupUnitServerWithDocs(t)
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
	ts, _ := setupUnitServerWithDocs(t)
	defer ts.Close()

	_, _ = ts.store.Create("Ticket 1", "body", "", nil, nil, []string{"api", "bug"})
	_, _ = ts.store.Create("Ticket 2", "body", "", nil, nil, []string{"API", "feature"})
	_, _ = ts.store.Create("Ticket 3", "body", "", nil, nil, []string{"bug"})

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

func TestTags_MergedTicketAndDocTags(t *testing.T) {
	ts, docsStore := setupUnitServerWithDocs(t)
	defer ts.Close()

	_, _ = ts.store.Create("Ticket 1", "body", "", nil, nil, []string{"api"})
	_, _ = docsStore.Create("Doc 1", "guides", "body", []string{"api", "docs"}, nil)

	resp := ts.makeRequest(t, http.MethodGet, "/tags", nil)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusOK)

	result := decode[ListTagsResponse](t, resp)
	if len(result.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d: %+v", len(result.Tags), result.Tags)
	}

	// api appears in both ticket and doc → count 2
	if result.Tags[0].Name != "api" || result.Tags[0].Count != 2 {
		t.Errorf("expected first tag api:2, got %s:%d", result.Tags[0].Name, result.Tags[0].Count)
	}
	if result.Tags[1].Name != "docs" || result.Tags[1].Count != 1 {
		t.Errorf("expected second tag docs:1, got %s:%d", result.Tags[1].Name, result.Tags[1].Count)
	}
}

func TestTags_NilDocsStoreManager(t *testing.T) {
	// Uses the standard setupUnitServer which doesn't set DocsStoreManager
	ts := setupUnitServer(t)
	defer ts.Close()

	_, _ = ts.store.Create("Ticket 1", "body", "", nil, nil, []string{"api"})

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
