package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/kareemaly/cortex/internal/events"
	"github.com/kareemaly/cortex/internal/ticket"
)

// --- Test helpers (redefined here since integration_test.go uses a build tag) ---

// unitServer wraps httptest.Server with test dependencies.
type unitServer struct {
	*httptest.Server
	store       *ticket.Store
	projectRoot string
}

func writeUnitConfig(t *testing.T, projectRoot string, repos map[string]string) {
	t.Helper()

	var b strings.Builder
	b.WriteString("name: test\n")
	if len(repos) == 0 {
		b.WriteString("repos: {}\n")
	} else {
		b.WriteString("repos:\n")
		keys := make([]string, 0, len(repos))
		for key := range repos {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			b.WriteString("  ")
			b.WriteString(key)
			b.WriteString(": ")
			b.WriteString(repos[key])
			b.WriteString("\n")
		}
	}

	if err := os.WriteFile(filepath.Join(projectRoot, "cortex.yaml"), []byte(b.String()), 0644); err != nil {
		t.Fatal(err)
	}
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
	req.Header.Set(ArchitectHeader, us.projectRoot)

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

func runGitCmd(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test User",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test User",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
	return string(out)
}

func createGitRepoWithCommit(t *testing.T) (string, string) {
	t.Helper()

	repoDir := t.TempDir()
	runGitCmd(t, repoDir, "init")
	if err := os.WriteFile(filepath.Join(repoDir, "file.txt"), []byte("hello\nworld\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGitCmd(t, repoDir, "add", "file.txt")
	runGitCmd(t, repoDir, "commit", "-m", "initial commit")
	sha := strings.TrimSpace(runGitCmd(t, repoDir, "rev-parse", "HEAD"))
	return repoDir, sha
}

func createGitRepoWithStructuredCommit(t *testing.T) (string, string) {
	t.Helper()

	repoDir := t.TempDir()
	runGitCmd(t, repoDir, "init")
	if err := os.WriteFile(filepath.Join(repoDir, "file.txt"), []byte("line1\nline2\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "delete.txt"), []byte("remove me\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGitCmd(t, repoDir, "add", "file.txt", "delete.txt")
	runGitCmd(t, repoDir, "commit", "-m", "base commit")

	if err := os.WriteFile(filepath.Join(repoDir, "file.txt"), []byte("line1\nline2 changed\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "added.txt"), []byte("brand new\ncontent\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(repoDir, "delete.txt")); err != nil {
		t.Fatal(err)
	}
	runGitCmd(t, repoDir, "add", "-A")
	runGitCmd(t, repoDir, "commit", "-m", "structured diff commit")
	sha := strings.TrimSpace(runGitCmd(t, repoDir, "rev-parse", "HEAD"))
	return repoDir, sha
}

// --- Test server setup ---

// setupUnitServer creates a test server suitable for unit tests (no build tag required).
// Creates a cortex.yaml at project root so ProjectRequired middleware passes.
func setupUnitServer(t *testing.T) *unitServer {
	t.Helper()

	tmpDir := t.TempDir()

	writeUnitConfig(t, tmpDir, nil)

	ticketsDir := filepath.Join(tmpDir, "tickets")
	store, err := ticket.NewStore(ticketsDir, nil, "")
	if err != nil {
		t.Fatalf("failed to create ticket store: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bus := events.NewBus()
	storeManager := NewStoreManager(logger, nil)
	storeManager.stores[tmpDir] = store

	sessionManager := NewSessionManager(logger)

	deps := &Dependencies{
		StoreManager:   storeManager,
		SessionManager: sessionManager,
		TmuxManager:    nil,
		Bus:            bus,
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

	created, _ := ts.store.Create("Due Date Ticket", "body", nil, nil, "")

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

	created, _ := ts.store.Create("Ticket", "body", nil, nil, "")

	req, _ := http.NewRequest(http.MethodPatch, ts.URL+"/tickets/"+created.ID+"/due-date", bytes.NewReader([]byte("bad json")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(ArchitectHeader, ts.projectRoot)

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

	created, _ := ts.store.Create("Ticket", "body", nil, nil, "")

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

	created, _ := ts.store.Create("Ticket", "body", nil, nil, "")

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

	created, _ := ts.store.Create("Ticket", "body", nil, nil, "")

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

	created, _ := ts.store.Create("Test Ticket", "body", nil, nil, "")

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

func TestGetDiffs_Success(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	repoDir, sha := createGitRepoWithStructuredCommit(t)
	writeUnitConfig(t, ts.projectRoot, map[string]string{"repo": repoDir})
	created, _ := ts.store.Create("Diff Ticket", "body", nil, nil, "repo")
	meta := &ticket.TicketConclusionMeta{
		StartedAt:   time.Now().UTC().Add(-time.Minute),
		ConcludedAt: time.Now().UTC(),
		Agent:       "codex",
		Commits:     []string{sha},
	}
	if err := ts.store.WriteConclusion(created.ID, meta, "done"); err != nil {
		t.Fatalf("failed to write conclusion: %v", err)
	}

	resp := ts.makeRequest(t, http.MethodGet, "/tickets/"+created.ID+"/diffs", nil)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusOK)

	result := decode[DiffsResponse](t, resp)
	if result.TicketID != created.ID {
		t.Fatalf("expected ticket ID %q, got %q", created.ID, result.TicketID)
	}
	if result.Repo != repoDir {
		t.Fatalf("expected repo %q, got %q", repoDir, result.Repo)
	}
	if len(result.Commits) != 1 {
		t.Fatalf("expected 1 commit diff, got %d", len(result.Commits))
	}
	commit := result.Commits[0]
	if commit.SHA != sha {
		t.Fatalf("expected SHA %q, got %q", sha, commit.SHA)
	}
	if commit.Subject != "structured diff commit" {
		t.Fatalf("expected subject %q, got %q", "structured diff commit", commit.Subject)
	}
	if commit.AuthorName != "Test User" {
		t.Fatalf("expected author name %q, got %q", "Test User", commit.AuthorName)
	}
	if len(commit.Files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(commit.Files))
	}

	byPath := make(map[string]DiffFileResponse, len(commit.Files))
	for _, file := range commit.Files {
		byPath[file.Path] = file
	}

	added := byPath["added.txt"]
	if added.Status != "added" {
		t.Fatalf("expected added.txt status added, got %q", added.Status)
	}
	if added.After == nil || *added.After != "brand new\ncontent\n" {
		t.Fatalf("unexpected added.txt after content: %#v", added.After)
	}
	if added.Before != nil {
		t.Fatalf("expected added.txt before to be nil")
	}
	if !strings.Contains(added.Patch, "+++ b/added.txt") {
		t.Fatalf("expected added.txt patch header, got %q", added.Patch)
	}

	deleted := byPath["delete.txt"]
	if deleted.Status != "deleted" {
		t.Fatalf("expected delete.txt status deleted, got %q", deleted.Status)
	}
	if deleted.Before == nil || *deleted.Before != "remove me\n" {
		t.Fatalf("unexpected delete.txt before content: %#v", deleted.Before)
	}
	if deleted.After != nil {
		t.Fatalf("expected delete.txt after to be nil")
	}

	modified := byPath["file.txt"]
	if modified.Status != "modified" {
		t.Fatalf("expected file.txt status modified, got %q", modified.Status)
	}
	if modified.Additions != 1 || modified.Deletions != 1 {
		t.Fatalf("expected file.txt additions/deletions to be 1/1, got %d/%d", modified.Additions, modified.Deletions)
	}
	if modified.Before != nil || modified.After != nil {
		t.Fatalf("expected modified file snapshots to be nil")
	}
	if !strings.Contains(modified.Patch, "@@") {
		t.Fatalf("expected file.txt patch hunks, got %q", modified.Patch)
	}
}

func TestGetDiffs_NoConclusion(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	created, _ := ts.store.Create("Diff Ticket", "body", nil, nil, "")
	resp := ts.makeRequest(t, http.MethodGet, "/tickets/"+created.ID+"/diffs", nil)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusNotFound)

	result := decode[ErrorResponse](t, resp)
	if result.Code != "no_conclusion" {
		t.Fatalf("expected code no_conclusion, got %q", result.Code)
	}
}

func TestGetDiffs_EmptyCommits(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	repoDir, _ := createGitRepoWithCommit(t)
	writeUnitConfig(t, ts.projectRoot, map[string]string{"repo": repoDir})
	created, _ := ts.store.Create("Diff Ticket", "body", nil, nil, "repo")
	meta := &ticket.TicketConclusionMeta{
		StartedAt:   time.Now().UTC().Add(-time.Minute),
		ConcludedAt: time.Now().UTC(),
		Agent:       "codex",
	}
	if err := ts.store.WriteConclusion(created.ID, meta, "done"); err != nil {
		t.Fatalf("failed to write conclusion: %v", err)
	}

	resp := ts.makeRequest(t, http.MethodGet, "/tickets/"+created.ID+"/diffs", nil)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusOK)

	result := decode[DiffsResponse](t, resp)
	if len(result.Commits) != 0 {
		t.Fatalf("expected 0 commit diffs, got %d", len(result.Commits))
	}
}

func TestGetDiffs_InvalidRepo(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	repoDir := filepath.Join(t.TempDir(), "missing")
	writeUnitConfig(t, ts.projectRoot, map[string]string{"repo": repoDir})
	created, _ := ts.store.Create("Diff Ticket", "body", nil, nil, "repo")
	meta := &ticket.TicketConclusionMeta{
		StartedAt:   time.Now().UTC().Add(-time.Minute),
		ConcludedAt: time.Now().UTC(),
		Agent:       "codex",
		Commits:     []string{"abc123"},
	}
	if err := ts.store.WriteConclusion(created.ID, meta, "done"); err != nil {
		t.Fatalf("failed to write conclusion: %v", err)
	}

	resp := ts.makeRequest(t, http.MethodGet, "/tickets/"+created.ID+"/diffs", nil)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusBadRequest)

	result := decode[ErrorResponse](t, resp)
	if result.Code != "invalid_repo" {
		t.Fatalf("expected code invalid_repo, got %q", result.Code)
	}
}

func TestGetDiffs_MissingCommit(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	repoDir, _ := createGitRepoWithCommit(t)
	writeUnitConfig(t, ts.projectRoot, map[string]string{"repo": repoDir})
	created, _ := ts.store.Create("Diff Ticket", "body", nil, nil, "repo")
	meta := &ticket.TicketConclusionMeta{
		StartedAt:   time.Now().UTC().Add(-time.Minute),
		ConcludedAt: time.Now().UTC(),
		Agent:       "codex",
		Commits:     []string{"deadbeef"},
	}
	if err := ts.store.WriteConclusion(created.ID, meta, "done"); err != nil {
		t.Fatalf("failed to write conclusion: %v", err)
	}

	resp := ts.makeRequest(t, http.MethodGet, "/tickets/"+created.ID+"/diffs", nil)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusNotFound)

	result := decode[ErrorResponse](t, resp)
	if result.Code != "commit_not_found" {
		t.Fatalf("expected code commit_not_found, got %q", result.Code)
	}
}

// --- Conclude ---

func TestConclude_Success(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	created, _ := ts.store.Create("Conclude Ticket", "body", nil, nil, "")

	// Use rejected=true to avoid needing a real git repo in the unit test.
	body := ConcludeSessionRequest{
		Content:         "done report",
		Rejected:        true,
		RejectionReason: "unit test — no git repo available",
	}
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

func TestConclude_MissingCommits(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	created, _ := ts.store.Create("Conclude Ticket", "body", nil, nil, "")

	body := ConcludeSessionRequest{Content: "done report"}
	resp := ts.makeRequest(t, http.MethodPost, "/tickets/"+created.ID+"/conclude", body)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusBadRequest)

	result := decode[ErrorResponse](t, resp)
	if result.Code != "validation_error" {
		t.Errorf("expected code 'validation_error', got %q", result.Code)
	}
}

func TestConclude_Rejected_NoReason(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	created, _ := ts.store.Create("Conclude Ticket", "body", nil, nil, "")

	body := ConcludeSessionRequest{Content: "done report", Rejected: true}
	resp := ts.makeRequest(t, http.MethodPost, "/tickets/"+created.ID+"/conclude", body)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusBadRequest)

	result := decode[ErrorResponse](t, resp)
	if result.Code != "validation_error" {
		t.Errorf("expected code 'validation_error', got %q", result.Code)
	}
}

func TestConclude_InvalidJSON(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	created, _ := ts.store.Create("Ticket", "body", nil, nil, "")

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/tickets/"+created.ID+"/conclude", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(ArchitectHeader, ts.projectRoot)

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

	created, _ := ts.store.Create("Ticket", "body", nil, nil, "")

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

	// Write cortex.yaml at root
	if err := os.WriteFile(filepath.Join(tmpDir, "cortex.yaml"), []byte("name: test\n"), 0644); err != nil {
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
		Bus:            events.NewBus(),
		Logger:         logger,
	}

	srv := httptest.NewServer(NewRouter(deps, deps.Logger))
	defer srv.Close()

	created, _ := store.Create("Focus Ticket", "body", nil, nil, "")

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/tickets/"+created.ID+"/focus", nil)
	req.Header.Set(ArchitectHeader, tmpDir)
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

	created, _ := ts.store.Create("Focus Ticket", "body", nil, nil, "")

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
	req.Header.Set(ArchitectHeader, ts.projectRoot)

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

	_, _ = ts.store.Create("Alpha Ticket", "body1", nil, nil, "")
	_, _ = ts.store.Create("Beta Ticket", "body2", nil, nil, "")

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/tickets?query=alpha", nil)
	req.Header.Set(ArchitectHeader, ts.projectRoot)

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

// --- Create edge cases ---

func TestCreate_WithDueDate(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	body := map[string]any{
		"title":    "Due Date Ticket",
		"body":     "body",
		"type":     "work",
		"repo":     "test-repo",
		"due_date": "2025-12-01T00:00:00Z",
	}
	writeUnitConfig(t, ts.projectRoot, map[string]string{"test-repo": ts.projectRoot})

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
	writeUnitConfig(t, ts.projectRoot, map[string]string{"test-repo": ts.projectRoot})

	body := map[string]any{
		"title":    "Bad Due Date",
		"body":     "body",
		"repo":     "test-repo",
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

func TestCreate_MissingRepo(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	body := map[string]any{
		"title": "Missing Repo",
		"body":  "body",
	}

	resp := ts.makeRequest(t, http.MethodPost, "/tickets", body)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusBadRequest)

	result := decode[ErrorResponse](t, resp)
	if result.Code != "missing_repo" {
		t.Errorf("expected code 'missing_repo', got %q", result.Code)
	}
}

// --- ListByStatus with due_before filter ---

func TestListByStatus_DueBeforeInvalidFormat(t *testing.T) {
	ts := setupUnitServer(t)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/tickets/backlog?due_before=bad", nil)
	req.Header.Set(ArchitectHeader, ts.projectRoot)

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
		{"review", false},
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
