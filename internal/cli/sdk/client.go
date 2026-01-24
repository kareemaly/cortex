package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
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

// HealthResponse is the response from the health endpoint.
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// TicketSummary is a brief view of a ticket.
type TicketSummary struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	Status           string    `json:"status"`
	Created          time.Time `json:"created"`
	HasActiveSession bool      `json:"has_active_session"`
	AgentStatus      *string   `json:"agent_status,omitempty"`
	AgentTool        *string   `json:"agent_tool,omitempty"`
}

// ListAllTicketsResponse groups tickets by status.
type ListAllTicketsResponse struct {
	Backlog  []TicketSummary `json:"backlog"`
	Progress []TicketSummary `json:"progress"`
	Review   []TicketSummary `json:"review"`
	Done     []TicketSummary `json:"done"`
}

// ListTicketsResponse is a list of tickets with a single status.
type ListTicketsResponse struct {
	Tickets []TicketSummary `json:"tickets"`
}

// DatesResponse is the dates portion of a ticket response.
type DatesResponse struct {
	Created  time.Time  `json:"created"`
	Updated  time.Time  `json:"updated"`
	Progress *time.Time `json:"progress,omitempty"`
	Reviewed *time.Time `json:"reviewed,omitempty"`
	Done     *time.Time `json:"done,omitempty"`
}

// CommentResponse is a comment in a ticket response.
type CommentResponse struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id,omitempty"`
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// StatusEntryResponse is a status entry in a session response.
type StatusEntryResponse struct {
	Status string    `json:"status"`
	Tool   *string   `json:"tool,omitempty"`
	Work   *string   `json:"work,omitempty"`
	At     time.Time `json:"at"`
}

// SessionResponse is a session in a ticket response.
type SessionResponse struct {
	ID            string                `json:"id"`
	StartedAt     time.Time             `json:"started_at"`
	EndedAt       *time.Time            `json:"ended_at,omitempty"`
	Agent         string                `json:"agent"`
	TmuxWindow    string                `json:"tmux_window"`
	CurrentStatus *StatusEntryResponse  `json:"current_status,omitempty"`
	StatusHistory []StatusEntryResponse `json:"status_history"`
}

// ArchitectSessionResponse is a session response for architect endpoints.
type ArchitectSessionResponse struct {
	ID          string     `json:"id"`
	TmuxSession string     `json:"tmux_session"`
	TmuxWindow  string     `json:"tmux_window"`
	StartedAt   time.Time  `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at,omitempty"`
}

// ArchitectStateResponse is the response from GET /architect.
type ArchitectStateResponse struct {
	State   string                    `json:"state"`
	Session *ArchitectSessionResponse `json:"session,omitempty"`
}

// ArchitectSpawnResponse is the response from POST /architect/spawn.
type ArchitectSpawnResponse struct {
	State       string                   `json:"state"`
	Session     ArchitectSessionResponse `json:"session"`
	TmuxSession string                   `json:"tmux_session"`
	TmuxWindow  string                   `json:"tmux_window"`
}

// TicketResponse is the full ticket response with status.
type TicketResponse struct {
	ID       string            `json:"id"`
	Title    string            `json:"title"`
	Body     string            `json:"body"`
	Status   string            `json:"status"`
	Dates    DatesResponse     `json:"dates"`
	Comments []CommentResponse `json:"comments"`
	Session  *SessionResponse  `json:"session,omitempty"`
}

// ErrorResponse is the standard error response format.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Details string `json:"details,omitempty"`
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
