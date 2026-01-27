package sdk

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kareemaly/cortex/internal/types"
)

const defaultBaseURL = "http://localhost:4200"

// ProjectHeader is the HTTP header name for specifying the project path.
const ProjectHeader = "X-Cortex-Project"

// Client is an HTTP client for communicating with the cortex daemon.
type Client struct {
	baseURL     string
	httpClient  *http.Client
	projectPath string
}

// NewClient creates a new client with the specified base URL and project path.
func NewClient(baseURL, projectPath string) *Client {
	return &Client{
		baseURL:     baseURL,
		projectPath: projectPath,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// DefaultClient returns a client configured for the default daemon address.
func DefaultClient(projectPath string) *Client {
	return NewClient(defaultBaseURL, projectPath)
}

// doRequest executes an HTTP request with the project header.
func (c *Client) doRequest(req *http.Request) (*http.Response, error) {
	if c.projectPath != "" {
		req.Header.Set(ProjectHeader, c.projectPath)
	}
	return c.httpClient.Do(req)
}

// Re-export shared types for SDK consumers
type (
	ErrorResponse            = types.ErrorResponse
	DatesResponse            = types.DatesResponse
	CommentResponse          = types.CommentResponse
	StatusEntryResponse      = types.StatusEntryResponse
	RequestedReviewResponse  = types.RequestedReviewResponse
	SessionResponse          = types.SessionResponse
	TicketResponse           = types.TicketResponse
	TicketSummary            = types.TicketSummary
	ListTicketsResponse      = types.ListTicketsResponse
	ListAllTicketsResponse   = types.ListAllTicketsResponse
	ArchitectSessionResponse = types.ArchitectSessionResponse
	ArchitectStateResponse   = types.ArchitectStateResponse
	ArchitectSpawnResponse   = types.ArchitectSpawnResponse
)

// HealthResponse is the response from the health endpoint.
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// APIError represents an error response from the API with its code preserved.
type APIError struct {
	Code    string
	Message string
	Status  int
}

func (e *APIError) Error() string {
	return e.Message
}

// IsOrphanedSession returns true if this error indicates an orphaned session.
func (e *APIError) IsOrphanedSession() bool {
	return e.Code == "session_orphaned"
}

// Health checks if the daemon is healthy.
func (c *Client) Health() error {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req) // Health doesn't need project header
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("daemon returned status %d", resp.StatusCode)
	}

	return nil
}

// HealthWithVersion checks daemon health and returns version info.
func (c *Client) HealthWithVersion() (*HealthResponse, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req) // Health doesn't need project header
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("daemon returned status %d", resp.StatusCode)
	}

	var health HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &health, nil
}

// ListAllTickets returns all tickets grouped by status.
// If query is non-empty, filters tickets by title or body (case-insensitive).
func (c *Client) ListAllTickets(query string) (*ListAllTicketsResponse, error) {
	url := c.baseURL + "/tickets"
	if query != "" {
		url += "?query=" + query
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result ListAllTicketsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ListTicketsByStatus returns tickets with a specific status.
// If query is non-empty, filters tickets by title or body (case-insensitive).
func (c *Client) ListTicketsByStatus(status, query string) (*ListTicketsResponse, error) {
	url := c.baseURL + "/tickets/" + status
	if query != "" {
		url += "?query=" + query
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result ListTicketsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetTicket returns a specific ticket by status and ID.
func (c *Client) GetTicket(status, id string) (*TicketResponse, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/tickets/"+status+"/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result TicketResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// CreateTicket creates a new ticket.
func (c *Client) CreateTicket(title, body string) (*TicketResponse, error) {
	reqBody := map[string]string{"title": title, "body": body}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/tickets", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		return nil, c.parseError(resp)
	}

	var result TicketResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetArchitect returns the current architect state.
func (c *Client) GetArchitect() (*ArchitectStateResponse, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/architect", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result ArchitectStateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// SpawnArchitect spawns or reattaches to an architect session.
func (c *Client) SpawnArchitect(mode string) (*ArchitectSpawnResponse, error) {
	url := c.baseURL + "/architect/spawn"
	if mode != "" {
		url += "?mode=" + mode
	}

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.parseError(resp)
	}

	var result ArchitectSpawnResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// SpawnSession spawns a new session for a ticket.
func (c *Client) SpawnSession(status, id, mode string) (*SessionResponse, error) {
	url := c.baseURL + "/tickets/" + status + "/" + id + "/spawn"
	if mode != "" {
		url += "?mode=" + mode
	}

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.parseError(resp)
	}

	var result SessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// KillSession kills a session by ID.
func (c *Client) KillSession(id string) error {
	req, err := http.NewRequest(http.MethodDelete, c.baseURL+"/sessions/"+id, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}

	return nil
}

// ApproveSession sends an approve prompt to an active session.
func (c *Client) ApproveSession(id string) error {
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/sessions/"+id+"/approve", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}

	return nil
}

// FindTicketByID searches for a ticket by ID across all statuses.
func (c *Client) FindTicketByID(ticketID string) (*TicketResponse, error) {
	all, err := c.ListAllTickets("")
	if err != nil {
		return nil, err
	}

	// Search through all statuses
	for _, summary := range all.Backlog {
		if summary.ID == ticketID || hasPrefix(summary.ID, ticketID) {
			return c.GetTicket("backlog", summary.ID)
		}
	}
	for _, summary := range all.Progress {
		if summary.ID == ticketID || hasPrefix(summary.ID, ticketID) {
			return c.GetTicket("progress", summary.ID)
		}
	}
	for _, summary := range all.Review {
		if summary.ID == ticketID || hasPrefix(summary.ID, ticketID) {
			return c.GetTicket("review", summary.ID)
		}
	}
	for _, summary := range all.Done {
		if summary.ID == ticketID || hasPrefix(summary.ID, ticketID) {
			return c.GetTicket("done", summary.ID)
		}
	}

	return nil, fmt.Errorf("ticket not found: %s", ticketID)
}

// ProjectTicketCounts holds ticket counts by status.
type ProjectTicketCounts struct {
	Backlog  int `json:"backlog"`
	Progress int `json:"progress"`
	Review   int `json:"review"`
	Done     int `json:"done"`
}

// ProjectResponse represents a single project in the API response.
type ProjectResponse struct {
	Path   string               `json:"path"`
	Title  string               `json:"title"`
	Exists bool                 `json:"exists"`
	Counts *ProjectTicketCounts `json:"counts,omitempty"`
}

// ListProjectsResponse is the response from GET /projects.
type ListProjectsResponse struct {
	Projects []ProjectResponse `json:"projects"`
}

// ListProjects returns all registered projects from the daemon.
func (c *Client) ListProjects() (*ListProjectsResponse, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/projects", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req) // No project header needed
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result ListProjectsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// AddCommentResponse is the response from the add comment endpoint.
type AddCommentResponse struct {
	Success bool            `json:"success"`
	Comment CommentResponse `json:"comment"`
}

// RequestReviewResponse is the response from the request review endpoint.
type RequestReviewResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	ReviewCount int    `json:"review_count"`
}

// ConcludeSessionResponse is the response from the conclude session endpoint.
type ConcludeSessionResponse struct {
	Success  bool   `json:"success"`
	TicketID string `json:"ticket_id"`
	Message  string `json:"message"`
}

// UpdateTicket updates a ticket's title and/or body by ID (status-agnostic).
func (c *Client) UpdateTicket(id string, title, body *string) (*TicketResponse, error) {
	// Discover current status
	current, err := c.GetTicketByID(id)
	if err != nil {
		return nil, err
	}

	reqBody := map[string]*string{}
	if title != nil {
		reqBody["title"] = title
	}
	if body != nil {
		reqBody["body"] = body
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPut, c.baseURL+"/tickets/"+current.Status+"/"+id, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result TicketResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// DeleteTicket deletes a ticket by ID (status-agnostic).
func (c *Client) DeleteTicket(id string) error {
	// Discover current status
	current, err := c.GetTicketByID(id)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodDelete, c.baseURL+"/tickets/"+current.Status+"/"+id, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent {
		return c.parseError(resp)
	}

	return nil
}

// MoveTicket moves a ticket to a different status by ID (status-agnostic).
func (c *Client) MoveTicket(id, toStatus string) (*TicketResponse, error) {
	// Discover current status
	current, err := c.GetTicketByID(id)
	if err != nil {
		return nil, err
	}

	reqBody := map[string]string{"to": toStatus}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/tickets/"+current.Status+"/"+id+"/move", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result TicketResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetTicketByID returns a ticket by ID regardless of status.
func (c *Client) GetTicketByID(id string) (*TicketResponse, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/tickets/by-id/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result TicketResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// AddComment adds a comment to a ticket.
func (c *Client) AddComment(ticketID, commentType, title, content string) (*AddCommentResponse, error) {
	reqBody := map[string]string{"type": commentType, "title": title, "content": content}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/tickets/"+ticketID+"/comments", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result AddCommentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// RequestReview requests a review for a ticket.
func (c *Client) RequestReview(ticketID, repoPath, title, content string) (*RequestReviewResponse, error) {
	reqBody := map[string]string{"repo_path": repoPath, "title": title, "content": content}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/tickets/"+ticketID+"/reviews", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result RequestReviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ConcludeSession concludes a ticket session.
func (c *Client) ConcludeSession(ticketID, fullReport string) (*ConcludeSessionResponse, error) {
	reqBody := map[string]string{"full_report": fullReport}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/tickets/"+ticketID+"/conclude", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result ConcludeSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// FocusDaemonDashboard focuses the CortexDaemon dashboard window.
func (c *Client) FocusDaemonDashboard() error {
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/daemon/focus", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req) // No project header needed
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}

	return nil
}

// FocusArchitect focuses the architect tmux window (window 0) for the project.
func (c *Client) FocusArchitect() error {
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/architect/focus", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}

	return nil
}

// FocusTicket focuses the tmux window of a ticket's active session.
func (c *Client) FocusTicket(ticketID string) error {
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/tickets/"+ticketID+"/focus", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}

	return nil
}

// Event represents an SSE event from the daemon.
type Event struct {
	Type        string `json:"type"`
	ProjectPath string `json:"project_path"`
	TicketID    string `json:"ticket_id"`
	Payload     any    `json:"payload,omitempty"`
}

// SubscribeEvents opens an SSE connection and returns a channel of events.
// The channel is closed when the context is cancelled or the connection drops.
func (c *Client) SubscribeEvents(ctx context.Context) (<-chan Event, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/events", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	if c.projectPath != "" {
		req.Header.Set(ProjectHeader, c.projectPath)
	}

	// Use a dedicated client with no timeout for the long-lived SSE connection.
	sseClient := &http.Client{}
	resp, err := sseClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to event stream: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("event stream returned status %d", resp.StatusCode)
	}

	ch := make(chan Event, 64)

	go func() {
		defer func() { _ = resp.Body.Close() }()
		defer close(ch)

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			var event Event
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}
			select {
			case ch <- event:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, nil
}

// parseError extracts an error message from a non-OK response.
func (c *Client) parseError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &APIError{
			Message: fmt.Sprintf("request failed with status %d", resp.StatusCode),
			Status:  resp.StatusCode,
		}
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		return &APIError{
			Message: fmt.Sprintf("request failed with status %d: %s", resp.StatusCode, string(body)),
			Status:  resp.StatusCode,
		}
	}

	message := errResp.Details
	if message == "" {
		message = errResp.Error
	}
	if message == "" {
		message = fmt.Sprintf("request failed with status %d", resp.StatusCode)
	}

	return &APIError{
		Code:    errResp.Code,
		Message: message,
		Status:  resp.StatusCode,
	}
}

// hasPrefix checks if id starts with prefix (for short ID matching).
func hasPrefix(id, prefix string) bool {
	return len(prefix) > 0 && len(id) >= len(prefix) && id[:len(prefix)] == prefix
}
