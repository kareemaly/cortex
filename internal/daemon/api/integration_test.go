//go:build integration

package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/kareemaly/cortex/internal/ticket"
)

// testServer wraps httptest.Server with test dependencies.
type testServer struct {
	*httptest.Server
	store       *ticket.Store
	projectRoot string
}

// setupTestServer creates a new test server with a temporary ticket store.
func setupTestServer(t *testing.T) *testServer {
	t.Helper()

	tmpDir := t.TempDir()
	ticketsDir := filepath.Join(tmpDir, ".cortex", "tickets")

	store, err := ticket.NewStore(ticketsDir)
	if err != nil {
		t.Fatalf("failed to create ticket store: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	storeManager := NewStoreManager(logger)
	// Pre-populate the store manager with our test store
	storeManager.stores[tmpDir] = store

	deps := &Dependencies{
		StoreManager: storeManager,
		TmuxManager:  nil, // tmux not used in tests
		HookExecutor: nil,
		Logger:       logger,
	}

	return &testServer{
		Server:      httptest.NewServer(NewRouter(deps, deps.Logger)),
		store:       store,
		projectRoot: tmpDir,
	}
}

// request makes an HTTP request to the test server.
func (ts *testServer) request(t *testing.T, method, path string, body any) *http.Response {
	t.Helper()

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, ts.URL+path, reqBody)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Add project header for non-health requests
	if path != "/health" {
		req.Header.Set(ProjectHeader, ts.projectRoot)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	return resp
}

// decodeJSON decodes a JSON response body into the specified type.
func decodeJSON[T any](t *testing.T, resp *http.Response) T {
	t.Helper()

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	return result
}

// expectStatus asserts the response has the expected status code.
func expectStatus(t *testing.T, resp *http.Response, expected int) {
	t.Helper()

	if resp.StatusCode != expected {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status %d, got %d: %s", expected, resp.StatusCode, string(body))
	}
}

// Test cases

func TestHealthEndpoint(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	resp := ts.request(t, http.MethodGet, "/health", nil)
	defer resp.Body.Close()

	expectStatus(t, resp, http.StatusOK)

	result := decodeJSON[HealthResponse](t, resp)
	if result.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", result.Status)
	}
}

func TestListAllTicketsEmpty(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	resp := ts.request(t, http.MethodGet, "/tickets", nil)
	defer resp.Body.Close()

	expectStatus(t, resp, http.StatusOK)

	result := decodeJSON[ListAllTicketsResponse](t, resp)
	if len(result.Backlog) != 0 {
		t.Errorf("expected 0 backlog tickets, got %d", len(result.Backlog))
	}
	if len(result.Progress) != 0 {
		t.Errorf("expected 0 progress tickets, got %d", len(result.Progress))
	}
	if len(result.Review) != 0 {
		t.Errorf("expected 0 review tickets, got %d", len(result.Review))
	}
	if len(result.Done) != 0 {
		t.Errorf("expected 0 done tickets, got %d", len(result.Done))
	}
}

func TestListAllTicketsWithData(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	// Create tickets in different statuses
	_, _ = ts.store.Create("Backlog Ticket", "body1")
	ticket2, _ := ts.store.Create("Progress Ticket", "body2")
	ticket3, _ := ts.store.Create("Review Ticket", "body3")
	ticket4, _ := ts.store.Create("Done Ticket", "body4")

	_ = ts.store.Move(ticket2.ID, ticket.StatusProgress)
	_ = ts.store.Move(ticket3.ID, ticket.StatusReview)
	_ = ts.store.Move(ticket4.ID, ticket.StatusDone)

	resp := ts.request(t, http.MethodGet, "/tickets", nil)
	defer resp.Body.Close()

	expectStatus(t, resp, http.StatusOK)

	result := decodeJSON[ListAllTicketsResponse](t, resp)
	if len(result.Backlog) != 1 {
		t.Errorf("expected 1 backlog ticket, got %d", len(result.Backlog))
	}
	if len(result.Progress) != 1 {
		t.Errorf("expected 1 progress ticket, got %d", len(result.Progress))
	}
	if len(result.Review) != 1 {
		t.Errorf("expected 1 review ticket, got %d", len(result.Review))
	}
	if len(result.Done) != 1 {
		t.Errorf("expected 1 done ticket, got %d", len(result.Done))
	}
}

func TestCreateTicket(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	body := CreateTicketRequest{
		Title: "New API Ticket",
		Body:  "Created via API",
	}

	resp := ts.request(t, http.MethodPost, "/tickets", body)
	defer resp.Body.Close()

	expectStatus(t, resp, http.StatusCreated)

	result := decodeJSON[TicketResponse](t, resp)
	if result.ID == "" {
		t.Error("expected ticket ID to be set")
	}
	if result.Title != "New API Ticket" {
		t.Errorf("expected title 'New API Ticket', got %q", result.Title)
	}
	if result.Body != "Created via API" {
		t.Errorf("expected body 'Created via API', got %q", result.Body)
	}
	if result.Status != "backlog" {
		t.Errorf("expected status 'backlog', got %q", result.Status)
	}
}

func TestCreateTicketInvalidJSON(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/tickets", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(ProjectHeader, ts.projectRoot)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	expectStatus(t, resp, http.StatusBadRequest)

	result := decodeJSON[ErrorResponse](t, resp)
	if result.Code != "invalid_json" {
		t.Errorf("expected code 'invalid_json', got %q", result.Code)
	}
}

func TestGetTicket(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	created, _ := ts.store.Create("Test Get Ticket", "Test body")

	resp := ts.request(t, http.MethodGet, "/tickets/backlog/"+created.ID, nil)
	defer resp.Body.Close()

	expectStatus(t, resp, http.StatusOK)

	result := decodeJSON[TicketResponse](t, resp)
	if result.ID != created.ID {
		t.Errorf("expected ID %q, got %q", created.ID, result.ID)
	}
	if result.Title != "Test Get Ticket" {
		t.Errorf("expected title 'Test Get Ticket', got %q", result.Title)
	}
}

func TestGetTicketNotFound(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	resp := ts.request(t, http.MethodGet, "/tickets/backlog/nonexistent-id", nil)
	defer resp.Body.Close()

	expectStatus(t, resp, http.StatusNotFound)

	result := decodeJSON[ErrorResponse](t, resp)
	if result.Code != "not_found" {
		t.Errorf("expected code 'not_found', got %q", result.Code)
	}
}

func TestGetTicketWrongStatus(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	// Create ticket in backlog
	created, _ := ts.store.Create("Backlog Ticket", "body")

	// Try to get it from progress
	resp := ts.request(t, http.MethodGet, "/tickets/progress/"+created.ID, nil)
	defer resp.Body.Close()

	expectStatus(t, resp, http.StatusNotFound)

	result := decodeJSON[ErrorResponse](t, resp)
	if result.Code != "not_found" {
		t.Errorf("expected code 'not_found', got %q", result.Code)
	}
}

func TestUpdateTicket(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	created, _ := ts.store.Create("Original Title", "Original body")

	newTitle := "Updated Title"
	newBody := "Updated body"
	body := UpdateTicketRequest{
		Title: &newTitle,
		Body:  &newBody,
	}

	resp := ts.request(t, http.MethodPut, "/tickets/backlog/"+created.ID, body)
	defer resp.Body.Close()

	expectStatus(t, resp, http.StatusOK)

	result := decodeJSON[TicketResponse](t, resp)
	if result.Title != "Updated Title" {
		t.Errorf("expected title 'Updated Title', got %q", result.Title)
	}
	if result.Body != "Updated body" {
		t.Errorf("expected body 'Updated body', got %q", result.Body)
	}
}

func TestDeleteTicket(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	created, _ := ts.store.Create("To Be Deleted", "body")

	resp := ts.request(t, http.MethodDelete, "/tickets/backlog/"+created.ID, nil)
	defer resp.Body.Close()

	expectStatus(t, resp, http.StatusNoContent)

	// Verify ticket is gone
	_, _, err := ts.store.Get(created.ID)
	if err == nil {
		t.Error("expected ticket to be deleted")
	}
}

func TestMoveTicket(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	created, _ := ts.store.Create("Movable Ticket", "body")

	body := MoveTicketRequest{To: "progress"}

	resp := ts.request(t, http.MethodPost, "/tickets/backlog/"+created.ID+"/move", body)
	defer resp.Body.Close()

	expectStatus(t, resp, http.StatusOK)

	result := decodeJSON[TicketResponse](t, resp)
	if result.Status != "progress" {
		t.Errorf("expected status 'progress', got %q", result.Status)
	}

	// Verify in store
	_, status, _ := ts.store.Get(created.ID)
	if status != ticket.StatusProgress {
		t.Errorf("expected store status 'progress', got %q", status)
	}
}

func TestMoveTicketToReview(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	created, _ := ts.store.Create("Review Ticket", "body")
	_ = ts.store.Move(created.ID, ticket.StatusProgress)

	body := MoveTicketRequest{To: "review"}

	resp := ts.request(t, http.MethodPost, "/tickets/progress/"+created.ID+"/move", body)
	defer resp.Body.Close()

	expectStatus(t, resp, http.StatusOK)

	result := decodeJSON[TicketResponse](t, resp)
	if result.Status != "review" {
		t.Errorf("expected status 'review', got %q", result.Status)
	}

	// Verify in store
	_, status, _ := ts.store.Get(created.ID)
	if status != ticket.StatusReview {
		t.Errorf("expected store status 'review', got %q", status)
	}
}

func TestMoveTicketInvalidStatus(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	created, _ := ts.store.Create("Test Ticket", "body")

	body := MoveTicketRequest{To: "invalid"}

	resp := ts.request(t, http.MethodPost, "/tickets/backlog/"+created.ID+"/move", body)
	defer resp.Body.Close()

	expectStatus(t, resp, http.StatusBadRequest)

	result := decodeJSON[ErrorResponse](t, resp)
	if result.Code != "invalid_status" {
		t.Errorf("expected code 'invalid_status', got %q", result.Code)
	}
}

func TestListByStatus(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	// Create tickets
	ts.store.Create("Backlog 1", "body")
	ts.store.Create("Backlog 2", "body")
	ticket3, _ := ts.store.Create("Progress Ticket", "body")
	ts.store.Move(ticket3.ID, ticket.StatusProgress)

	resp := ts.request(t, http.MethodGet, "/tickets/backlog", nil)
	defer resp.Body.Close()

	expectStatus(t, resp, http.StatusOK)

	result := decodeJSON[ListTicketsResponse](t, resp)
	if len(result.Tickets) != 2 {
		t.Errorf("expected 2 backlog tickets, got %d", len(result.Tickets))
	}

	for _, tkt := range result.Tickets {
		if tkt.Status != "backlog" {
			t.Errorf("expected status 'backlog', got %q", tkt.Status)
		}
	}
}

func TestInvalidStatus(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	resp := ts.request(t, http.MethodGet, "/tickets/invalid", nil)
	defer resp.Body.Close()

	expectStatus(t, resp, http.StatusBadRequest)

	result := decodeJSON[ErrorResponse](t, resp)
	if result.Code != "invalid_status" {
		t.Errorf("expected code 'invalid_status', got %q", result.Code)
	}
}

func TestMissingProjectHeader(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	// Make request without project header
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/tickets", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	expectStatus(t, resp, http.StatusBadRequest)

	result := decodeJSON[ErrorResponse](t, resp)
	if result.Code != "missing_project_header" {
		t.Errorf("expected code 'missing_project_header', got %q", result.Code)
	}
}

func TestInvalidProjectPath(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	// Make request with relative path
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/tickets", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set(ProjectHeader, "relative/path")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	expectStatus(t, resp, http.StatusBadRequest)

	result := decodeJSON[ErrorResponse](t, resp)
	if result.Code != "invalid_project_path" {
		t.Errorf("expected code 'invalid_project_path', got %q", result.Code)
	}
}

func TestProjectNotFound(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	// Make request with non-existent path
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/tickets", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set(ProjectHeader, "/nonexistent/path/12345")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	expectStatus(t, resp, http.StatusNotFound)

	result := decodeJSON[ErrorResponse](t, resp)
	if result.Code != "project_not_found" {
		t.Errorf("expected code 'project_not_found', got %q", result.Code)
	}
}
