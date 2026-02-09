package sdk

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	daemonconfig "github.com/kareemaly/cortex/internal/daemon/config"
	"github.com/kareemaly/cortex/internal/types"
)

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
// It checks CORTEX_DAEMON_URL env var first, falling back to the default.
func DefaultClient(projectPath string) *Client {
	baseURL := os.Getenv("CORTEX_DAEMON_URL")
	if baseURL == "" {
		baseURL = daemonconfig.DefaultDaemonURL
	}
	return NewClient(baseURL, projectPath)
}

// WithProject returns a new Client targeting a different project.
// The new client shares the underlying HTTP client for efficiency.
// If projectPath is empty, returns the same client unchanged.
func (c *Client) WithProject(projectPath string) *Client {
	if projectPath == "" {
		return c
	}
	return &Client{
		baseURL:     c.baseURL,
		httpClient:  c.httpClient,
		projectPath: projectPath,
	}
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
	CommentResponse          = types.CommentResponse
	SessionResponse          = types.SessionResponse
	TicketResponse           = types.TicketResponse
	TicketSummary            = types.TicketSummary
	ListTicketsResponse      = types.ListTicketsResponse
	ListAllTicketsResponse   = types.ListAllTicketsResponse
	ArchitectSessionResponse = types.ArchitectSessionResponse
	ArchitectStateResponse   = types.ArchitectStateResponse
	ArchitectSpawnResponse   = types.ArchitectSpawnResponse
	DocResponse              = types.DocResponse
	DocSummary               = types.DocSummary
	ListDocsResponse         = types.ListDocsResponse
	HealthResponse           = types.HealthResponse
	ProjectTicketCounts      = types.ProjectTicketCounts
	ProjectResponse          = types.ProjectResponse
	AddCommentResponse       = types.AddCommentResponse
	RequestReviewResponse    = types.RequestReviewResponse
	ConcludeSessionResponse  = types.ConcludeSessionResponse
	ResolvePromptResponse    = types.ResolvePromptResponse
	ListTagsResponse         = types.ListTagsResponse
	TagCount                 = types.TagCount
	MetaSpawnResponse        = types.MetaSpawnResponse
	MetaStateResponse        = types.MetaStateResponse
)

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
// If dueBefore is non-nil, filters tickets with due date before the specified time.
// If tag is non-empty, filters tickets that have the specified tag.
func (c *Client) ListAllTickets(query string, dueBefore *time.Time, tag string) (*ListAllTicketsResponse, error) {
	url := c.baseURL + "/tickets"
	params := []string{}
	if query != "" {
		params = append(params, "query="+query)
	}
	if dueBefore != nil {
		params = append(params, "due_before="+dueBefore.Format(time.RFC3339))
	}
	if tag != "" {
		params = append(params, "tag="+tag)
	}
	if len(params) > 0 {
		url += "?" + strings.Join(params, "&")
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
// If dueBefore is non-nil, filters tickets with due date before the specified time.
// If tag is non-empty, filters tickets that have the specified tag.
func (c *Client) ListTicketsByStatus(status, query string, dueBefore *time.Time, tag string) (*ListTicketsResponse, error) {
	url := c.baseURL + "/tickets/" + status
	params := []string{}
	if query != "" {
		params = append(params, "query="+query)
	}
	if dueBefore != nil {
		params = append(params, "due_before="+dueBefore.Format(time.RFC3339))
	}
	if tag != "" {
		params = append(params, "tag="+tag)
	}
	if len(params) > 0 {
		url += "?" + strings.Join(params, "&")
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
func (c *Client) CreateTicket(title, body, ticketType string, dueDate *time.Time, references, tags []string) (*TicketResponse, error) {
	reqBody := map[string]any{"title": title, "body": body}
	if ticketType != "" {
		reqBody["type"] = ticketType
	}
	if dueDate != nil {
		reqBody["due_date"] = dueDate.Format(time.RFC3339)
	}
	if references != nil {
		reqBody["references"] = references
	}
	if tags != nil {
		reqBody["tags"] = tags
	}
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
	all, err := c.ListAllTickets("", nil, "")
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

// UnlinkProject removes a project from the global registry.
// This does not delete any files, only removes the project from tracking.
func (c *Client) UnlinkProject(projectPath string) error {
	req, err := http.NewRequest(http.MethodDelete, c.baseURL+"/projects?path="+projectPath, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req) // No project header needed
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent {
		return c.parseError(resp)
	}

	return nil
}

// UpdateTicket updates a ticket's title, body, references, and/or tags by ID (status-agnostic).
func (c *Client) UpdateTicket(id string, title, body *string, references, tags *[]string) (*TicketResponse, error) {
	// Discover current status
	current, err := c.GetTicketByID(id)
	if err != nil {
		return nil, err
	}

	reqBody := map[string]any{}
	if title != nil {
		reqBody["title"] = *title
	}
	if body != nil {
		reqBody["body"] = *body
	}
	if references != nil {
		reqBody["references"] = *references
	}
	if tags != nil {
		reqBody["tags"] = *tags
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

// SetDueDate sets the due date for a ticket.
func (c *Client) SetDueDate(ticketID string, dueDate time.Time) (*TicketResponse, error) {
	reqBody := map[string]string{"due_date": dueDate.Format(time.RFC3339)}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPatch, c.baseURL+"/tickets/"+ticketID+"/due-date", bytes.NewReader(jsonBody))
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

// ClearDueDate removes the due date from a ticket.
func (c *Client) ClearDueDate(ticketID string) (*TicketResponse, error) {
	req, err := http.NewRequest(http.MethodDelete, c.baseURL+"/tickets/"+ticketID+"/due-date", nil)
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
// If author is non-empty, it's sent explicitly; otherwise the API resolves from the session.
func (c *Client) AddComment(ticketID, commentType, content, author string) (*AddCommentResponse, error) {
	reqBody := map[string]string{"type": commentType, "content": content}
	if author != "" {
		reqBody["author"] = author
	}
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
func (c *Client) RequestReview(ticketID, repoPath, content, commit string) (*RequestReviewResponse, error) {
	reqBody := map[string]string{"repo_path": repoPath, "content": content}
	if commit != "" {
		reqBody["commit"] = commit
	}
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
func (c *Client) ConcludeSession(ticketID, content string) (*ConcludeSessionResponse, error) {
	reqBody := map[string]string{"content": content}
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

// ConcludeArchitectSession concludes the architect session.
func (c *Client) ConcludeArchitectSession(content string) (*ConcludeSessionResponse, error) {
	reqBody := map[string]string{"content": content}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/architect/conclude", bytes.NewReader(jsonBody))
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

// ExecuteCommentAction executes an action attached to a comment.
func (c *Client) ExecuteCommentAction(ticketID, commentID string) error {
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/tickets/"+ticketID+"/comments/"+commentID+"/execute", nil)
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

// ResolvePromptRequest is the request parameters for resolving a prompt.
type ResolvePromptRequest struct {
	Role       string // "architect" or "ticket"
	Stage      string // "SYSTEM", "KICKOFF", "APPROVE"
	TicketType string // for ticket prompts only
}

// ResolvePrompt resolves a prompt file with extension fallback.
func (c *Client) ResolvePrompt(req ResolvePromptRequest) (*ResolvePromptResponse, error) {
	url := c.baseURL + "/prompts/resolve?role=" + req.Role + "&stage=" + req.Stage
	if req.TicketType != "" {
		url += "&type=" + req.TicketType
	}

	httpReq, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.doRequest(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result ResolvePromptResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// CreateDoc creates a new doc.
func (c *Client) CreateDoc(title, category, body string, tags, references []string) (*DocResponse, error) {
	reqBody := map[string]any{
		"title":    title,
		"category": category,
		"body":     body,
	}
	if tags != nil {
		reqBody["tags"] = tags
	}
	if references != nil {
		reqBody["references"] = references
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/docs", bytes.NewReader(jsonBody))
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

	var docResult DocResponse
	if err := json.NewDecoder(resp.Body).Decode(&docResult); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &docResult, nil
}

// GetDoc returns a doc by ID.
func (c *Client) GetDoc(id string) (*DocResponse, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/docs/"+id, nil)
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

	var docResult DocResponse
	if err := json.NewDecoder(resp.Body).Decode(&docResult); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &docResult, nil
}

// UpdateDoc updates a doc's fields.
func (c *Client) UpdateDoc(id string, title, body *string, tags, references *[]string) (*DocResponse, error) {
	reqBody := map[string]any{}
	if title != nil {
		reqBody["title"] = *title
	}
	if body != nil {
		reqBody["body"] = *body
	}
	if tags != nil {
		reqBody["tags"] = *tags
	}
	if references != nil {
		reqBody["references"] = *references
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPut, c.baseURL+"/docs/"+id, bytes.NewReader(jsonBody))
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

	var docResult DocResponse
	if err := json.NewDecoder(resp.Body).Decode(&docResult); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &docResult, nil
}

// DeleteDoc deletes a doc by ID.
func (c *Client) DeleteDoc(id string) error {
	req, err := http.NewRequest(http.MethodDelete, c.baseURL+"/docs/"+id, nil)
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

// MoveDoc moves a doc to a different category.
func (c *Client) MoveDoc(id, category string) (*DocResponse, error) {
	reqBody := map[string]string{"category": category}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/docs/"+id+"/move", bytes.NewReader(jsonBody))
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

	var docResult DocResponse
	if err := json.NewDecoder(resp.Body).Decode(&docResult); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &docResult, nil
}

// ListDocs lists docs with optional filters.
func (c *Client) ListDocs(category, tag, query string) (*ListDocsResponse, error) {
	url := c.baseURL + "/docs"
	var params []string
	if category != "" {
		params = append(params, "category="+category)
	}
	if tag != "" {
		params = append(params, "tag="+tag)
	}
	if query != "" {
		params = append(params, "query="+query)
	}
	if len(params) > 0 {
		url += "?" + strings.Join(params, "&")
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

	var listResult ListDocsResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResult); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &listResult, nil
}

// ListTags returns aggregated tags from tickets and docs, sorted by count descending.
func (c *Client) ListTags() (*ListTagsResponse, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/tags", nil)
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

	var result ListTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// AddDocComment adds a comment to a doc.
func (c *Client) AddDocComment(docID, commentType, content, author string) (*AddCommentResponse, error) {
	reqBody := map[string]string{"type": commentType, "content": content}
	if author != "" {
		reqBody["author"] = author
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/docs/"+docID+"/comments", bytes.NewReader(jsonBody))
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

// SessionListItem represents a session in the list response.
type SessionListItem struct {
	SessionID   string    `json:"session_id"`
	SessionType string    `json:"session_type"`
	TicketID    string    `json:"ticket_id"`
	TicketTitle string    `json:"ticket_title"`
	Agent       string    `json:"agent"`
	TmuxWindow  string    `json:"tmux_window"`
	StartedAt   time.Time `json:"started_at"`
	Status      string    `json:"status"`
	Tool        *string   `json:"tool,omitempty"`
}

// ListSessionsResponse is the response from GET /sessions.
type ListSessionsResponse struct {
	Sessions []SessionListItem `json:"sessions"`
	Total    int               `json:"total"`
}

// ListSessions returns all active sessions for the project.
func (c *Client) ListSessions() (*ListSessionsResponse, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/sessions", nil)
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

	var result ListSessionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// SpawnMeta spawns or reattaches to the meta session.
func (c *Client) SpawnMeta(mode string) (*MetaSpawnResponse, error) {
	url := c.baseURL + "/meta/spawn"
	if mode != "" {
		url += "?mode=" + mode
	}

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req) // No project header needed
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.parseError(resp)
	}

	var result MetaSpawnResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetMetaState returns the current meta session state.
func (c *Client) GetMetaState() (*MetaStateResponse, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/meta", nil)
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

	var result MetaStateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ConcludeMetaSession concludes the meta session.
func (c *Client) ConcludeMetaSession(content string) (*ConcludeSessionResponse, error) {
	reqBody := map[string]string{"content": content}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/meta/conclude", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req) // No project header needed
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

// FocusMeta focuses the meta tmux window.
func (c *Client) FocusMeta() error {
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/meta/focus", nil)
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

// RegisterProject registers a project with the daemon.
func (c *Client) RegisterProject(path, title string) error {
	reqBody := map[string]string{"path": path}
	if title != "" {
		reqBody["title"] = title
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/projects", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req) // No project header needed
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}

	return nil
}
