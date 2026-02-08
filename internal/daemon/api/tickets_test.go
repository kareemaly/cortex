package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/kareemaly/cortex/internal/ticket"
)

// --- Test helpers (redefined here since integration_test.go uses a build tag) ---

// unitServer wraps httptest.Server with test dependencies.
type unitServer struct {
	*httptest.Server
	store       *ticket.Store
	projectRoot string
}

// makeRequest makes an HTTP request to the test server with the project header.
func (us *unitServer) makeRequest(t *testing.T, method, path string, body any) *http.Response {
	t.Helper()

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, us.URL+path, reqBody)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set(ProjectHeader, us.projectRoot)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}

// assertStatus asserts the response has the expected status code.
func assertStatus(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status %d, got %d: %s", expected, resp.StatusCode, string(body))
	}
}

// decode decodes a JSON response body into the specified type.
func decode[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	return result
}

// --- Test server setup ---

// setupUnitServer creates a test server suitable for unit tests (no build tag required).
// Creates a .cortex/ dir with cortex.yaml so ProjectRequired middleware passes.
func setupUnitServer(t *testing.T) *unitServer {
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

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	storeManager := NewStoreManager(logger, nil)
	storeManager.stores[tmpDir] = store

	sessionManager := NewSessionManager(logger)

	deps := &Dependencies{
		StoreManager:   storeManager,
		SessionManager: sessionManager,
		TmuxManager:    nil,
		Logger:         logger,
	}

	return &unitServer{
		Server:      httptest.NewServer(NewRouter(deps, deps.Logger)),
		store:       store,
		projectRoot: tmpDir,
	}
}

// --- SetDueDate ---

func TestSetDueDate_Success(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	created, _ := ts.store.Create("Due Date Ticket", "body", "", nil, nil, nil)

	body := SetDueDateRequest{DueDate: "2025-06-01T00:00:00Z"}
	resp := ts.makeRequest(t, http.MethodPatch, "/tickets/"+created.ID+"/due-date", body)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusOK)

	result := decode[TicketResponse](t, resp)
	if result.Due == nil {
		t.Error("expected due date to be set")
	}
}

func TestSetDueDate_InvalidJSON(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	created, _ := ts.store.Create("Ticket", "body", "", nil, nil, nil)

	req, _ := http.NewRequest(http.MethodPatch, ts.URL+"/tickets/"+created.ID+"/due-date", bytes.NewReader([]byte("bad json")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(ProjectHeader, ts.projectRoot)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusBadRequest)

	result := decode[ErrorResponse](t, resp)
	if result.Code != "invalid_json" {
		t.Errorf("expected code 'invalid_json', got %q", result.Code)
	}
}

func TestSetDueDate_EmptyDueDate(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	created, _ := ts.store.Create("Ticket", "body", "", nil, nil, nil)

	body := SetDueDateRequest{DueDate: ""}
	resp := ts.makeRequest(t, http.MethodPatch, "/tickets/"+created.ID+"/due-date", body)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusBadRequest)

	result := decode[ErrorResponse](t, resp)
	if result.Code != "validation_error" {
		t.Errorf("expected code 'validation_error', got %q", result.Code)
	}
}

func TestSetDueDate_InvalidFormat(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	created, _ := ts.store.Create("Ticket", "body", "", nil, nil, nil)

	body := SetDueDateRequest{DueDate: "not-a-date"}
	resp := ts.makeRequest(t, http.MethodPatch, "/tickets/"+created.ID+"/due-date", body)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusBadRequest)

	result := decode[ErrorResponse](t, resp)
	if result.Code != "invalid_due_date" {
		t.Errorf("expected code 'invalid_due_date', got %q", result.Code)
	}
}

func TestSetDueDate_NotFound(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	body := SetDueDateRequest{DueDate: "2025-06-01T00:00:00Z"}
	resp := ts.makeRequest(t, http.MethodPatch, "/tickets/nonexistent/due-date", body)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusNotFound)
}

// --- ClearDueDate ---

func TestClearDueDate_Success(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	created, _ := ts.store.Create("Ticket", "body", "", nil, nil, nil)

	resp := ts.makeRequest(t, http.MethodDelete, "/tickets/"+created.ID+"/due-date", nil)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusOK)

	result := decode[TicketResponse](t, resp)
	if result.Due != nil {
		t.Error("expected due date to be nil")
	}
}

func TestClearDueDate_NotFound(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	resp := ts.makeRequest(t, http.MethodDelete, "/tickets/nonexistent/due-date", nil)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusNotFound)
}

// --- GetByID ---

func TestGetByID_Success(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	created, _ := ts.store.Create("Test Ticket", "body", "", nil, nil, nil)

	resp := ts.makeRequest(t, http.MethodGet, "/tickets/by-id/"+created.ID, nil)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusOK)

	result := decode[TicketResponse](t, resp)
	if result.ID != created.ID {
		t.Errorf("expected ID %q, got %q", created.ID, result.ID)
	}
	if result.Status != "backlog" {
		t.Errorf("expected status 'backlog', got %q", result.Status)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	resp := ts.makeRequest(t, http.MethodGet, "/tickets/by-id/nonexistent", nil)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusNotFound)
}

// --- AddComment ---

func TestAddComment_Success(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	created, _ := ts.store.Create("Comment Ticket", "body", "", nil, nil, nil)

	body := AddCommentRequest{
		Type:    "comment",
		Content: "test comment",
		Author:  "test-agent",
	}
	resp := ts.makeRequest(t, http.MethodPost, "/tickets/"+created.ID+"/comments", body)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusOK)

	result := decode[AddCommentResponse](t, resp)
	if !result.Success {
		t.Error("expected success")
	}
	if result.Comment.Content != "test comment" {
		t.Errorf("expected content 'test comment', got %q", result.Comment.Content)
	}
	if result.Comment.Author != "test-agent" {
		t.Errorf("expected author 'test-agent', got %q", result.Comment.Author)
	}
}

func TestAddComment_InvalidJSON(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	created, _ := ts.store.Create("Ticket", "body", "", nil, nil, nil)

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/tickets/"+created.ID+"/comments", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(ProjectHeader, ts.projectRoot)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusBadRequest)
}

func TestAddComment_InvalidType(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	created, _ := ts.store.Create("Ticket", "body", "", nil, nil, nil)

	body := AddCommentRequest{
		Type:    "invalid_type",
		Content: "test",
	}
	resp := ts.makeRequest(t, http.MethodPost, "/tickets/"+created.ID+"/comments", body)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusBadRequest)

	result := decode[ErrorResponse](t, resp)
	if result.Code != "validation_error" {
		t.Errorf("expected code 'validation_error', got %q", result.Code)
	}
}

func TestAddComment_TicketNotFound(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	body := AddCommentRequest{
		Type:    "comment",
		Content: "test",
	}
	resp := ts.makeRequest(t, http.MethodPost, "/tickets/nonexistent/comments", body)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusNotFound)
}

func TestAddComment_AllValidTypes(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	validTypes := []string{"comment", "review_requested", "done", "blocker"}

	for _, ct := range validTypes {
		t.Run(ct, func(t *testing.T) {
			created, _ := ts.store.Create("Ticket for "+ct, "body", "", nil, nil, nil)

			body := AddCommentRequest{
				Type:    ct,
				Content: "test " + ct,
				Author:  "agent",
			}
			resp := ts.makeRequest(t, http.MethodPost, "/tickets/"+created.ID+"/comments", body)
			defer func() { _ = resp.Body.Close() }()

			assertStatus(t, resp, http.StatusOK)

			result := decode[AddCommentResponse](t, resp)
			if !result.Success {
				t.Errorf("expected success for type %q", ct)
			}
			if result.Comment.Type != ct {
				t.Errorf("expected type %q, got %q", ct, result.Comment.Type)
			}
		})
	}
}

func TestAddComment_WithExplicitAuthor(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	created, _ := ts.store.Create("Ticket", "body", "", nil, nil, nil)

	body := AddCommentRequest{
		Type:    "comment",
		Content: "test",
		Author:  "custom-author",
	}
	resp := ts.makeRequest(t, http.MethodPost, "/tickets/"+created.ID+"/comments", body)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusOK)

	result := decode[AddCommentResponse](t, resp)
	if result.Comment.Author != "custom-author" {
		t.Errorf("expected author 'custom-author', got %q", result.Comment.Author)
	}
}

// --- RequestReview ---

func TestRequestReview_Success(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	created, _ := ts.store.Create("Review Ticket", "body", "", nil, nil, nil)
	_ = ts.store.Move(created.ID, ticket.StatusProgress)

	body := RequestReviewRequest{
		RepoPath: ts.projectRoot,
		Content:  "please review",
	}
	resp := ts.makeRequest(t, http.MethodPost, "/tickets/"+created.ID+"/reviews", body)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusOK)

	result := decode[RequestReviewResponse](t, resp)
	if !result.Success {
		t.Error("expected success")
	}
}

func TestRequestReview_InvalidJSON(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	created, _ := ts.store.Create("Ticket", "body", "", nil, nil, nil)

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/tickets/"+created.ID+"/reviews", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(ProjectHeader, ts.projectRoot)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusBadRequest)
}

func TestRequestReview_EmptyRepoPath(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	created, _ := ts.store.Create("Ticket", "body", "", nil, nil, nil)

	body := RequestReviewRequest{
		RepoPath: "",
		Content:  "review this",
	}
	resp := ts.makeRequest(t, http.MethodPost, "/tickets/"+created.ID+"/reviews", body)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusBadRequest)

	result := decode[ErrorResponse](t, resp)
	if result.Code != "validation_error" {
		t.Errorf("expected code 'validation_error', got %q", result.Code)
	}
}

func TestRequestReview_EmptyContent(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	created, _ := ts.store.Create("Ticket", "body", "", nil, nil, nil)

	body := RequestReviewRequest{
		RepoPath: "/some/path",
		Content:  "",
	}
	resp := ts.makeRequest(t, http.MethodPost, "/tickets/"+created.ID+"/reviews", body)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusBadRequest)

	result := decode[ErrorResponse](t, resp)
	if result.Code != "validation_error" {
		t.Errorf("expected code 'validation_error', got %q", result.Code)
	}
}

func TestRequestReview_MovesToReview(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	created, _ := ts.store.Create("Ticket", "body", "", nil, nil, nil)
	_ = ts.store.Move(created.ID, ticket.StatusProgress)

	body := RequestReviewRequest{
		RepoPath: ts.projectRoot,
		Content:  "review please",
	}
	resp := ts.makeRequest(t, http.MethodPost, "/tickets/"+created.ID+"/reviews", body)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusOK)

	// Verify ticket moved to review
	_, status, _ := ts.store.Get(created.ID)
	if status != ticket.StatusReview {
		t.Errorf("expected status 'review', got %q", status)
	}
}

// --- Conclude ---

func TestConclude_Success(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	created, _ := ts.store.Create("Conclude Ticket", "body", "", nil, nil, nil)

	body := ConcludeSessionRequest{Content: "done report"}
	resp := ts.makeRequest(t, http.MethodPost, "/tickets/"+created.ID+"/conclude", body)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusOK)

	result := decode[ConcludeSessionResponse](t, resp)
	if !result.Success {
		t.Error("expected success")
	}
	if result.TicketID != created.ID {
		t.Errorf("expected ticket ID %q, got %q", created.ID, result.TicketID)
	}

	// Verify ticket moved to done
	_, status, _ := ts.store.Get(created.ID)
	if status != ticket.StatusDone {
		t.Errorf("expected status 'done', got %q", status)
	}
}

func TestConclude_InvalidJSON(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	created, _ := ts.store.Create("Ticket", "body", "", nil, nil, nil)

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/tickets/"+created.ID+"/conclude", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(ProjectHeader, ts.projectRoot)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusBadRequest)
}

func TestConclude_EmptyContent(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	created, _ := ts.store.Create("Ticket", "body", "", nil, nil, nil)

	body := ConcludeSessionRequest{Content: ""}
	resp := ts.makeRequest(t, http.MethodPost, "/tickets/"+created.ID+"/conclude", body)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusBadRequest)

	result := decode[ErrorResponse](t, resp)
	if result.Code != "validation_error" {
		t.Errorf("expected code 'validation_error', got %q", result.Code)
	}
}

func TestConclude_NotFound(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	body := ConcludeSessionRequest{Content: "done"}
	resp := ts.makeRequest(t, http.MethodPost, "/tickets/nonexistent/conclude", body)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusNotFound)
}

// --- Focus ---

func TestFocus_NoSessionManager(t *testing.T) {
	tmpDir := t.TempDir()
	cortexDir := filepath.Join(tmpDir, ".cortex")
	if err := os.MkdirAll(cortexDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cortexDir, "cortex.yaml"), []byte("name: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	ticketsDir := filepath.Join(tmpDir, "tickets")
	store, _ := ticket.NewStore(ticketsDir, nil, "")

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	storeManager := NewStoreManager(logger, nil)
	storeManager.stores[tmpDir] = store

	deps := &Dependencies{
		StoreManager:   storeManager,
		SessionManager: nil,
		Logger:         logger,
	}

	srv := httptest.NewServer(NewRouter(deps, deps.Logger))
	defer srv.Close()

	created, _ := store.Create("Focus Ticket", "body", "", nil, nil, nil)

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/tickets/"+created.ID+"/focus", nil)
	req.Header.Set(ProjectHeader, tmpDir)
	resp, _ := http.DefaultClient.Do(req)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusNotFound)

	result := decode[ErrorResponse](t, resp)
	if result.Code != "no_active_session" {
		t.Errorf("expected code 'no_active_session', got %q", result.Code)
	}
}

func TestFocus_NoActiveSession(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	created, _ := ts.store.Create("Focus Ticket", "body", "", nil, nil, nil)

	resp := ts.makeRequest(t, http.MethodPost, "/tickets/"+created.ID+"/focus", nil)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusNotFound)

	result := decode[ErrorResponse](t, resp)
	if result.Code != "no_active_session" {
		t.Errorf("expected code 'no_active_session', got %q", result.Code)
	}
}

// --- List filters ---

func TestListAll_DueBeforeInvalidFormat(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/tickets?due_before=not-a-date", nil)
	req.Header.Set(ProjectHeader, ts.projectRoot)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusBadRequest)

	result := decode[ErrorResponse](t, resp)
	if result.Code != "invalid_due_before" {
		t.Errorf("expected code 'invalid_due_before', got %q", result.Code)
	}
}

func TestListAll_WithQueryFilter(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	_, _ = ts.store.Create("Alpha Ticket", "body1", "", nil, nil, nil)
	_, _ = ts.store.Create("Beta Ticket", "body2", "", nil, nil, nil)

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/tickets?query=alpha", nil)
	req.Header.Set(ProjectHeader, ts.projectRoot)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusOK)

	result := decode[ListAllTicketsResponse](t, resp)
	if len(result.Backlog) != 1 {
		t.Errorf("expected 1 filtered ticket, got %d", len(result.Backlog))
	}
	if len(result.Backlog) > 0 && result.Backlog[0].Title != "Alpha Ticket" {
		t.Errorf("expected 'Alpha Ticket', got %q", result.Backlog[0].Title)
	}
}

func TestListAll_WithTagFilter(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	_, _ = ts.store.Create("Tagged Ticket", "body", "", nil, nil, []string{"important"})
	_, _ = ts.store.Create("Untagged Ticket", "body", "", nil, nil, nil)

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/tickets?tag=important", nil)
	req.Header.Set(ProjectHeader, ts.projectRoot)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusOK)

	result := decode[ListAllTicketsResponse](t, resp)
	if len(result.Backlog) != 1 {
		t.Errorf("expected 1 tagged ticket, got %d", len(result.Backlog))
	}
}

// --- Create edge cases ---

func TestCreate_WithDueDate(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	body := map[string]any{
		"title":    "Due Date Ticket",
		"body":     "body",
		"due_date": "2025-12-01T00:00:00Z",
	}

	resp := ts.makeRequest(t, http.MethodPost, "/tickets", body)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusCreated)

	result := decode[TicketResponse](t, resp)
	if result.Due == nil {
		t.Error("expected due date to be set")
	}
}

func TestCreate_InvalidDueDate(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	body := map[string]any{
		"title":    "Bad Due Date",
		"body":     "body",
		"due_date": "not-a-date",
	}

	resp := ts.makeRequest(t, http.MethodPost, "/tickets", body)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusBadRequest)

	result := decode[ErrorResponse](t, resp)
	if result.Code != "invalid_due_date" {
		t.Errorf("expected code 'invalid_due_date', got %q", result.Code)
	}
}

// --- ListByStatus with due_before filter ---

func TestListByStatus_DueBeforeInvalidFormat(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/tickets/backlog?due_before=bad", nil)
	req.Header.Set(ProjectHeader, ts.projectRoot)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusBadRequest)

	result := decode[ErrorResponse](t, resp)
	if result.Code != "invalid_due_before" {
		t.Errorf("expected code 'invalid_due_before', got %q", result.Code)
	}
}

// --- Helper function tests ---

func TestValidStatus(t *testing.T) {
	tests := []struct {
		status string
		valid  bool
	}{
		{"backlog", true},
		{"progress", true},
		{"review", true},
		{"done", true},
		{"invalid", false},
		{"", false},
		{"BACKLOG", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := validStatus(tt.status); got != tt.valid {
				t.Errorf("validStatus(%q) = %v, want %v", tt.status, got, tt.valid)
			}
		})
	}
}
