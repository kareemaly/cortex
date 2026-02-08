package sdk

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kareemaly/cortex/internal/types"
)

// --- Test helpers ---

// requestLog captures details of an incoming request.
type requestLog struct {
	Method  string
	Path    string
	Headers http.Header
	Body    []byte
}

// routedServer is a mock server that routes responses based on method+path.
type routedServer struct {
	mu       sync.Mutex
	routes   map[string]routeEntry
	requests []requestLog
}

type routeEntry struct {
	status int
	body   any
}

func newRoutedServer(t *testing.T) (*httptest.Server, *routedServer) {
	t.Helper()
	rs := &routedServer{routes: make(map[string]routeEntry)}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		rs.mu.Lock()
		rs.requests = append(rs.requests, requestLog{
			Method:  r.Method,
			Path:    r.URL.RequestURI(),
			Headers: r.Header.Clone(),
			Body:    body,
		})
		rs.mu.Unlock()

		key := r.Method + " " + r.URL.Path
		rs.mu.Lock()
		entry, ok := rs.routes[key]
		rs.mu.Unlock()
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(types.ErrorResponse{Error: "not found", Code: "not_found"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(entry.status)
		_ = json.NewEncoder(w).Encode(entry.body)
	}))
	t.Cleanup(srv.Close)
	return srv, rs
}

func (rs *routedServer) setRoute(method, path string, status int, body any) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.routes[method+" "+path] = routeEntry{status: status, body: body}
}

func (rs *routedServer) lastRequest() requestLog {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if len(rs.requests) == 0 {
		return requestLog{}
	}
	return rs.requests[len(rs.requests)-1]
}

// --- Constructor / Config tests ---

func TestNewClient(t *testing.T) {
	c := NewClient("http://example.com", "/my/project")
	if c.baseURL != "http://example.com" {
		t.Errorf("expected baseURL %q, got %q", "http://example.com", c.baseURL)
	}
	if c.projectPath != "/my/project" {
		t.Errorf("expected projectPath %q, got %q", "/my/project", c.projectPath)
	}
}

func TestDefaultClient(t *testing.T) {
	c := DefaultClient("/my/project")
	if c.baseURL != defaultBaseURL {
		t.Errorf("expected baseURL %q, got %q", defaultBaseURL, c.baseURL)
	}
}

func TestWithProject(t *testing.T) {
	c := NewClient("http://example.com", "/project-a")

	c2 := c.WithProject("/project-b")
	if c2.projectPath != "/project-b" {
		t.Errorf("expected projectPath %q, got %q", "/project-b", c2.projectPath)
	}
	if c2.baseURL != c.baseURL {
		t.Errorf("expected shared baseURL")
	}
}

func TestWithProjectSharesHTTPClient(t *testing.T) {
	c := NewClient("http://example.com", "/project-a")
	c2 := c.WithProject("/project-b")
	if c2.httpClient != c.httpClient {
		t.Error("expected shared HTTP client")
	}
}

func TestWithProjectEmpty(t *testing.T) {
	c := NewClient("http://example.com", "/project-a")
	c2 := c.WithProject("")
	if c2 != c {
		t.Error("expected same client when projectPath is empty")
	}
}

// --- doRequest / project header tests ---

func TestDoRequest_InjectsProjectHeader(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("GET", "/test", http.StatusOK, map[string]string{"ok": "true"})

	c := NewClient(srv.URL, "/my/project")
	req, _ := http.NewRequest("GET", srv.URL+"/test", nil)
	resp, err := c.doRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	last := rs.lastRequest()
	if last.Headers.Get(ProjectHeader) != "/my/project" {
		t.Errorf("expected project header %q, got %q", "/my/project", last.Headers.Get(ProjectHeader))
	}
}

func TestDoRequest_EmptyProjectPath(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("GET", "/test", http.StatusOK, map[string]string{"ok": "true"})

	c := NewClient(srv.URL, "")
	req, _ := http.NewRequest("GET", srv.URL+"/test", nil)
	resp, err := c.doRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	last := rs.lastRequest()
	if last.Headers.Get(ProjectHeader) != "" {
		t.Errorf("expected empty project header, got %q", last.Headers.Get(ProjectHeader))
	}
}

// --- Health ---

func TestHealth_Success(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("GET", "/health", http.StatusOK, HealthResponse{Status: "ok", Version: "1.0.0"})

	c := NewClient(srv.URL, "/p")
	err := c.Health()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestHealth_ServerError(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("GET", "/health", http.StatusInternalServerError, nil)

	c := NewClient(srv.URL, "/p")
	err := c.Health()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected 500 in error, got %v", err)
	}
}

func TestHealth_ConnectionError(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "/p")
	c.httpClient.Timeout = 100 * time.Millisecond
	err := c.Health()
	if err == nil {
		t.Fatal("expected connection error")
	}
}

func TestHealthWithVersion_Success(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("GET", "/health", http.StatusOK, HealthResponse{Status: "ok", Version: "v1.2.3"})

	c := NewClient(srv.URL, "/p")
	resp, err := c.HealthWithVersion()
	if err != nil {
		t.Fatal(err)
	}
	if resp.Version != "v1.2.3" {
		t.Errorf("expected version v1.2.3, got %q", resp.Version)
	}
	if resp.Status != "ok" {
		t.Errorf("expected status ok, got %q", resp.Status)
	}
}

func TestHealthWithVersion_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "/p")
	_, err := c.HealthWithVersion()
	if err == nil {
		t.Fatal("expected error on bad JSON")
	}
}

// --- ListProjects ---

func TestListProjects_Success(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("GET", "/projects", http.StatusOK, ListProjectsResponse{
		Projects: []ProjectResponse{
			{Path: "/proj1", Title: "Proj 1", Exists: true},
		},
	})

	c := NewClient(srv.URL, "/p")
	resp, err := c.ListProjects()
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Projects) != 1 {
		t.Errorf("expected 1 project, got %d", len(resp.Projects))
	}
	if resp.Projects[0].Path != "/proj1" {
		t.Errorf("expected path /proj1, got %q", resp.Projects[0].Path)
	}
}

// --- UnlinkProject ---

func TestUnlinkProject_Success(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("DELETE", "/projects", http.StatusNoContent, nil)

	c := NewClient(srv.URL, "/p")
	err := c.UnlinkProject("/proj1")
	if err != nil {
		t.Fatal(err)
	}
	last := rs.lastRequest()
	if !strings.Contains(last.Path, "path=/proj1") {
		t.Errorf("expected path query param, got %q", last.Path)
	}
}

// --- FocusDaemonDashboard ---

func TestFocusDaemonDashboard_Success(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("POST", "/daemon/focus", http.StatusOK, map[string]bool{"success": true})

	c := NewClient(srv.URL, "/p")
	err := c.FocusDaemonDashboard()
	if err != nil {
		t.Fatal(err)
	}
}

// --- Ticket CRUD ---

func TestListAllTickets_NoFilters(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("GET", "/tickets", http.StatusOK, ListAllTicketsResponse{
		Backlog:  []TicketSummary{{ID: "t1", Title: "T1", Status: "backlog"}},
		Progress: []TicketSummary{},
		Review:   []TicketSummary{},
		Done:     []TicketSummary{},
	})

	c := NewClient(srv.URL, "/p")
	resp, err := c.ListAllTickets("", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Backlog) != 1 {
		t.Errorf("expected 1 backlog, got %d", len(resp.Backlog))
	}

	last := rs.lastRequest()
	if strings.Contains(last.Path, "?") {
		t.Errorf("expected no query params, got %q", last.Path)
	}
}

func TestListAllTickets_WithFilters(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("GET", "/tickets", http.StatusOK, ListAllTicketsResponse{})

	c := NewClient(srv.URL, "/p")
	due := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	_, err := c.ListAllTickets("search", &due, "bug")
	if err != nil {
		t.Fatal(err)
	}

	last := rs.lastRequest()
	if !strings.Contains(last.Path, "query=search") {
		t.Errorf("expected query param, got %q", last.Path)
	}
	if !strings.Contains(last.Path, "due_before=") {
		t.Errorf("expected due_before param, got %q", last.Path)
	}
	if !strings.Contains(last.Path, "tag=bug") {
		t.Errorf("expected tag param, got %q", last.Path)
	}
}

func TestListTicketsByStatus(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("GET", "/tickets/backlog", http.StatusOK, ListTicketsResponse{
		Tickets: []TicketSummary{{ID: "t1", Status: "backlog"}},
	})

	c := NewClient(srv.URL, "/p")
	resp, err := c.ListTicketsByStatus("backlog", "", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Tickets) != 1 {
		t.Errorf("expected 1 ticket, got %d", len(resp.Tickets))
	}
}

func TestGetTicket_Success(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("GET", "/tickets/backlog/abc123", http.StatusOK, TicketResponse{
		ID:     "abc123",
		Title:  "Test",
		Status: "backlog",
	})

	c := NewClient(srv.URL, "/p")
	resp, err := c.GetTicket("backlog", "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if resp.ID != "abc123" {
		t.Errorf("expected ID abc123, got %q", resp.ID)
	}
}

func TestGetTicket_NotFound(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("GET", "/tickets/backlog/missing", http.StatusNotFound, types.ErrorResponse{
		Error: "not found",
		Code:  "not_found",
	})

	c := NewClient(srv.URL, "/p")
	_, err := c.GetTicket("backlog", "missing")
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Status != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", apiErr.Status)
	}
}

func TestGetTicketByID(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("GET", "/tickets/by-id/abc123", http.StatusOK, TicketResponse{
		ID:     "abc123",
		Title:  "Test",
		Status: "backlog",
	})

	c := NewClient(srv.URL, "/p")
	resp, err := c.GetTicketByID("abc123")
	if err != nil {
		t.Fatal(err)
	}
	if resp.ID != "abc123" {
		t.Errorf("expected ID abc123, got %q", resp.ID)
	}
}

func TestCreateTicket_Success(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("POST", "/tickets", http.StatusCreated, TicketResponse{
		ID:     "new1",
		Title:  "New Ticket",
		Status: "backlog",
	})

	c := NewClient(srv.URL, "/p")
	resp, err := c.CreateTicket("New Ticket", "body", "", nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.ID != "new1" {
		t.Errorf("expected ID new1, got %q", resp.ID)
	}

	last := rs.lastRequest()
	if last.Headers.Get("Content-Type") != "application/json" {
		t.Errorf("expected content-type application/json, got %q", last.Headers.Get("Content-Type"))
	}
}

func TestCreateTicket_WithAllFields(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("POST", "/tickets", http.StatusCreated, TicketResponse{ID: "new2"})

	c := NewClient(srv.URL, "/p")
	due := time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)
	_, err := c.CreateTicket("T", "B", "bug", &due, []string{"ref1"}, []string{"tag1"})
	if err != nil {
		t.Fatal(err)
	}

	last := rs.lastRequest()
	var body map[string]any
	_ = json.Unmarshal(last.Body, &body)
	if body["type"] != "bug" {
		t.Errorf("expected type bug, got %v", body["type"])
	}
	if body["due_date"] == nil {
		t.Error("expected due_date in body")
	}
}

func TestCreateTicket_Error(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("POST", "/tickets", http.StatusBadRequest, types.ErrorResponse{
		Error: "title required",
		Code:  "validation_error",
	})

	c := NewClient(srv.URL, "/p")
	_, err := c.CreateTicket("", "", "", nil, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdateTicket_Success(t *testing.T) {
	srv, rs := newRoutedServer(t)
	// GetTicketByID first
	rs.setRoute("GET", "/tickets/by-id/abc123", http.StatusOK, TicketResponse{
		ID:     "abc123",
		Status: "backlog",
	})
	// Then PUT
	rs.setRoute("PUT", "/tickets/backlog/abc123", http.StatusOK, TicketResponse{
		ID:    "abc123",
		Title: "Updated",
	})

	c := NewClient(srv.URL, "/p")
	title := "Updated"
	resp, err := c.UpdateTicket("abc123", &title, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Title != "Updated" {
		t.Errorf("expected title Updated, got %q", resp.Title)
	}
}

func TestDeleteTicket_Success(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("GET", "/tickets/by-id/abc123", http.StatusOK, TicketResponse{
		ID:     "abc123",
		Status: "backlog",
	})
	rs.setRoute("DELETE", "/tickets/backlog/abc123", http.StatusNoContent, nil)

	c := NewClient(srv.URL, "/p")
	err := c.DeleteTicket("abc123")
	if err != nil {
		t.Fatal(err)
	}
}

func TestMoveTicket_Success(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("GET", "/tickets/by-id/abc123", http.StatusOK, TicketResponse{
		ID:     "abc123",
		Status: "backlog",
	})
	rs.setRoute("POST", "/tickets/backlog/abc123/move", http.StatusOK, TicketResponse{
		ID:     "abc123",
		Status: "progress",
	})

	c := NewClient(srv.URL, "/p")
	resp, err := c.MoveTicket("abc123", "progress")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "progress" {
		t.Errorf("expected status progress, got %q", resp.Status)
	}
}

// --- Due dates ---

func TestSetDueDate_Success(t *testing.T) {
	srv, rs := newRoutedServer(t)
	due := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	rs.setRoute("PATCH", "/tickets/abc123/due-date", http.StatusOK, TicketResponse{
		ID:  "abc123",
		Due: &due,
	})

	c := NewClient(srv.URL, "/p")
	resp, err := c.SetDueDate("abc123", due)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Due == nil {
		t.Error("expected due date to be set")
	}
}

func TestClearDueDate_Success(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("DELETE", "/tickets/abc123/due-date", http.StatusOK, TicketResponse{
		ID: "abc123",
	})

	c := NewClient(srv.URL, "/p")
	resp, err := c.ClearDueDate("abc123")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Due != nil {
		t.Error("expected due date to be nil")
	}
}

// --- Sessions ---

func TestSpawnSession_Success(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("POST", "/tickets/backlog/abc123/spawn", http.StatusCreated, SessionResponse{
		Type:       "ticket",
		TicketID:   "abc123",
		Agent:      "claude-code",
		TmuxWindow: "win-1",
	})

	c := NewClient(srv.URL, "/p")
	resp, err := c.SpawnSession("backlog", "abc123", "")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Agent != "claude-code" {
		t.Errorf("expected agent claude-code, got %q", resp.Agent)
	}
}

func TestKillSession_Success(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("DELETE", "/sessions/sess1", http.StatusNoContent, nil)

	c := NewClient(srv.URL, "/p")
	err := c.KillSession("sess1")
	if err != nil {
		t.Fatal(err)
	}
}

func TestApproveSession_Success(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("POST", "/sessions/sess1/approve", http.StatusOK, map[string]bool{"ok": true})

	c := NewClient(srv.URL, "/p")
	err := c.ApproveSession("sess1")
	if err != nil {
		t.Fatal(err)
	}
}

func TestListSessions_Success(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("GET", "/sessions", http.StatusOK, ListSessionsResponse{
		Sessions: []SessionListItem{{SessionID: "s1", Agent: "claude"}},
		Total:    1,
	})

	c := NewClient(srv.URL, "/p")
	resp, err := c.ListSessions()
	if err != nil {
		t.Fatal(err)
	}
	if resp.Total != 1 {
		t.Errorf("expected total 1, got %d", resp.Total)
	}
}

// --- Architect ---

func TestGetArchitect_Success(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("GET", "/architect", http.StatusOK, ArchitectStateResponse{
		State: "none",
	})

	c := NewClient(srv.URL, "/p")
	resp, err := c.GetArchitect()
	if err != nil {
		t.Fatal(err)
	}
	if resp.State != "none" {
		t.Errorf("expected state none, got %q", resp.State)
	}
}

func TestSpawnArchitect_Success(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("POST", "/architect/spawn", http.StatusCreated, ArchitectSpawnResponse{
		State:       "active",
		TmuxSession: "cortex",
		TmuxWindow:  "architect",
	})

	c := NewClient(srv.URL, "/p")
	resp, err := c.SpawnArchitect("")
	if err != nil {
		t.Fatal(err)
	}
	if resp.State != "active" {
		t.Errorf("expected state active, got %q", resp.State)
	}
}

func TestConcludeArchitectSession(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("POST", "/architect/conclude", http.StatusOK, ConcludeSessionResponse{
		Success: true,
		Message: "concluded",
	})

	c := NewClient(srv.URL, "/p")
	resp, err := c.ConcludeArchitectSession("summary")
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Success {
		t.Error("expected success")
	}
}

// --- Comments / Reviews ---

func TestAddComment_Success(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("POST", "/tickets/abc123/comments", http.StatusOK, AddCommentResponse{
		Success: true,
		Comment: CommentResponse{ID: "c1", Type: "comment", Content: "test"},
	})

	c := NewClient(srv.URL, "/p")
	resp, err := c.AddComment("abc123", "comment", "test", "agent")
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Success {
		t.Error("expected success")
	}
	if resp.Comment.ID != "c1" {
		t.Errorf("expected comment ID c1, got %q", resp.Comment.ID)
	}
}

func TestRequestReview_Success(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("POST", "/tickets/abc123/reviews", http.StatusOK, RequestReviewResponse{
		Success: true,
		Message: "review requested",
	})

	c := NewClient(srv.URL, "/p")
	resp, err := c.RequestReview("abc123", "/repo", "review this", "abc")
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Success {
		t.Error("expected success")
	}

	last := rs.lastRequest()
	var body map[string]string
	_ = json.Unmarshal(last.Body, &body)
	if body["repo_path"] != "/repo" {
		t.Errorf("expected repo_path /repo, got %q", body["repo_path"])
	}
	if body["commit"] != "abc" {
		t.Errorf("expected commit abc, got %q", body["commit"])
	}
}

func TestConcludeSession_Success(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("POST", "/tickets/abc123/conclude", http.StatusOK, ConcludeSessionResponse{
		Success:  true,
		TicketID: "abc123",
		Message:  "concluded",
	})

	c := NewClient(srv.URL, "/p")
	resp, err := c.ConcludeSession("abc123", "done report")
	if err != nil {
		t.Fatal(err)
	}
	if resp.TicketID != "abc123" {
		t.Errorf("expected ticket ID abc123, got %q", resp.TicketID)
	}
}

func TestExecuteCommentAction(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("POST", "/tickets/t1/comments/c1/execute", http.StatusOK, map[string]bool{"success": true})

	c := NewClient(srv.URL, "/p")
	err := c.ExecuteCommentAction("t1", "c1")
	if err != nil {
		t.Fatal(err)
	}

	last := rs.lastRequest()
	if last.Method != "POST" {
		t.Errorf("expected POST, got %q", last.Method)
	}
}

// --- Docs ---

func TestCreateDoc(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("POST", "/docs", http.StatusCreated, DocResponse{
		ID:       "d1",
		Title:    "Doc",
		Category: "guides",
	})

	c := NewClient(srv.URL, "/p")
	resp, err := c.CreateDoc("Doc", "guides", "body", []string{"tag1"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.ID != "d1" {
		t.Errorf("expected ID d1, got %q", resp.ID)
	}
}

func TestGetDoc(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("GET", "/docs/d1", http.StatusOK, DocResponse{
		ID:    "d1",
		Title: "Doc",
	})

	c := NewClient(srv.URL, "/p")
	resp, err := c.GetDoc("d1")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Title != "Doc" {
		t.Errorf("expected title Doc, got %q", resp.Title)
	}
}

func TestUpdateDoc(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("PUT", "/docs/d1", http.StatusOK, DocResponse{
		ID:    "d1",
		Title: "Updated",
	})

	c := NewClient(srv.URL, "/p")
	title := "Updated"
	resp, err := c.UpdateDoc("d1", &title, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Title != "Updated" {
		t.Errorf("expected title Updated, got %q", resp.Title)
	}
}

func TestDeleteDoc(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("DELETE", "/docs/d1", http.StatusNoContent, nil)

	c := NewClient(srv.URL, "/p")
	err := c.DeleteDoc("d1")
	if err != nil {
		t.Fatal(err)
	}
}

func TestMoveDoc(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("POST", "/docs/d1/move", http.StatusOK, DocResponse{
		ID:       "d1",
		Category: "archive",
	})

	c := NewClient(srv.URL, "/p")
	resp, err := c.MoveDoc("d1", "archive")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Category != "archive" {
		t.Errorf("expected category archive, got %q", resp.Category)
	}
}

func TestListDocs(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("GET", "/docs", http.StatusOK, ListDocsResponse{
		Docs: []DocSummary{{ID: "d1", Title: "Doc"}},
	})

	c := NewClient(srv.URL, "/p")
	resp, err := c.ListDocs("guides", "tag1", "search")
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Docs) != 1 {
		t.Errorf("expected 1 doc, got %d", len(resp.Docs))
	}

	last := rs.lastRequest()
	if !strings.Contains(last.Path, "category=guides") {
		t.Errorf("expected category param, got %q", last.Path)
	}
	if !strings.Contains(last.Path, "tag=tag1") {
		t.Errorf("expected tag param, got %q", last.Path)
	}
	if !strings.Contains(last.Path, "query=search") {
		t.Errorf("expected query param, got %q", last.Path)
	}
}

func TestAddDocComment(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("POST", "/docs/d1/comments", http.StatusOK, AddCommentResponse{
		Success: true,
		Comment: CommentResponse{ID: "c1"},
	})

	c := NewClient(srv.URL, "/p")
	resp, err := c.AddDocComment("d1", "comment", "content", "author")
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Success {
		t.Error("expected success")
	}
}

// --- Error parsing ---

func TestParseError_WithDetails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(types.ErrorResponse{
			Error:   "bad request",
			Code:    "validation_error",
			Details: "title is required",
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "/p")
	_, err := c.GetTicket("backlog", "x")
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Code != "validation_error" {
		t.Errorf("expected code validation_error, got %q", apiErr.Code)
	}
	// Details takes precedence
	if apiErr.Message != "title is required" {
		t.Errorf("expected message 'title is required', got %q", apiErr.Message)
	}
}

func TestParseError_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("gateway error"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "/p")
	_, err := c.GetTicket("backlog", "x")
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Status != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", apiErr.Status)
	}
	if !strings.Contains(apiErr.Message, "gateway error") {
		t.Errorf("expected raw body in message, got %q", apiErr.Message)
	}
}

func TestAPIError_IsOrphanedSession(t *testing.T) {
	e := &APIError{Code: "session_orphaned", Message: "orphaned"}
	if !e.IsOrphanedSession() {
		t.Error("expected IsOrphanedSession() == true")
	}

	e2 := &APIError{Code: "other_code", Message: "other"}
	if e2.IsOrphanedSession() {
		t.Error("expected IsOrphanedSession() == false")
	}
}

// --- Focus ---

func TestFocusArchitect_Success(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("POST", "/architect/focus", http.StatusOK, map[string]bool{"ok": true})

	c := NewClient(srv.URL, "/p")
	err := c.FocusArchitect()
	if err != nil {
		t.Fatal(err)
	}
}

func TestFocusTicket_Success(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("POST", "/tickets/t1/focus", http.StatusOK, map[string]bool{"ok": true})

	c := NewClient(srv.URL, "/p")
	err := c.FocusTicket("t1")
	if err != nil {
		t.Fatal(err)
	}
}

// --- ResolvePrompt ---

func TestResolvePrompt_Success(t *testing.T) {
	srv, rs := newRoutedServer(t)
	rs.setRoute("GET", "/prompts/resolve", http.StatusOK, ResolvePromptResponse{
		Content:    "prompt content",
		SourcePath: "/path/to/prompt.md",
	})

	c := NewClient(srv.URL, "/p")
	resp, err := c.ResolvePrompt(ResolvePromptRequest{
		Role:  "architect",
		Stage: "SYSTEM",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "prompt content" {
		t.Errorf("expected content, got %q", resp.Content)
	}

	last := rs.lastRequest()
	if !strings.Contains(last.Path, "role=architect") {
		t.Errorf("expected role param, got %q", last.Path)
	}
	if !strings.Contains(last.Path, "stage=SYSTEM") {
		t.Errorf("expected stage param, got %q", last.Path)
	}
}

// --- hasPrefix helper ---

func TestHasPrefix(t *testing.T) {
	tests := []struct {
		id     string
		prefix string
		want   bool
	}{
		{"abc123", "abc", true},
		{"abc123", "abc123", true},
		{"abc123", "xyz", false},
		{"abc", "abcdef", false},
		{"abc", "", false},
		{"", "abc", false},
	}
	for _, tt := range tests {
		got := hasPrefix(tt.id, tt.prefix)
		if got != tt.want {
			t.Errorf("hasPrefix(%q, %q) = %v, want %v", tt.id, tt.prefix, got, tt.want)
		}
	}
}
