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

// ArchitectHeader is the HTTP header name for specifying the architect path.
const ArchitectHeader = "X-Cortex-Architect"

// Client is an HTTP client for communicating with the cortex daemon.
type Client struct {
	baseURL     string
	httpClient  *http.Client
	architectPath string
}

// NewClient creates a new client with the specified base URL and project path.
func NewClient(baseURL, architectPath string) *Client {
	return &Client{
		baseURL:     baseURL,
		architectPath: architectPath,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// DefaultClient returns a client configured for the default daemon address.
// It checks CORTEX_DAEMON_URL env var first, falling back to the default.
func DefaultClient(architectPath string) *Client {
	baseURL := os.Getenv("CORTEX_DAEMON_URL")
	if baseURL == "" {
		baseURL = daemonconfig.DefaultDaemonURL
	}
	return NewClient(baseURL, architectPath)
}

// doRequest executes an HTTP request with the project header.
func (c *Client) doRequest(req *http.Request) (*http.Response, error) {
	if c.architectPath != "" {
		req.Header.Set(ArchitectHeader, c.architectPath)
	}
	return c.httpClient.Do(req)
}

// Re-export shared types for SDK consumers
type (
	ErrorResponse            = types.ErrorResponse
	SessionResponse          = types.SessionResponse
	TicketResponse           = types.TicketResponse
	TicketSummary            = types.TicketSummary
	ListTicketsResponse      = types.ListTicketsResponse
	ListAllTicketsResponse   = types.ListAllTicketsResponse
	ArchitectSessionResponse = types.ArchitectSessionResponse
	ArchitectStateResponse   = types.ArchitectStateResponse
	ArchitectSpawnResponse   = types.ArchitectSpawnResponse
	HealthResponse           = types.HealthResponse
	ArchitectTicketCounts    = types.ArchitectTicketCounts
	ArchitectResponse        = types.ArchitectResponse
	ConcludeSessionResponse  = types.ConcludeSessionResponse
	ConclusionResponse       = types.ConclusionResponse
	ConclusionSummary        = types.ConclusionSummary
	ListConclusionsResponse  = types.ListConclusionsResponse
	ResolvePromptResponse    = types.ResolvePromptResponse
	PromptFileInfo           = types.PromptFileInfo
	PromptGroupInfo          = types.PromptGroupInfo
	ListPromptsResponse      = types.ListPromptsResponse
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
func (c *Client) ListAllTickets(query string, dueBefore *time.Time) (*ListAllTicketsResponse, error) {
	url := c.baseURL + "/tickets"
	params := []string{}
	if query != "" {
		params = append(params, "query="+query)
	}
	if dueBefore != nil {
		params = append(params, "due_before="+dueBefore.Format(time.RFC3339))
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
func (c *Client) ListTicketsByStatus(status, query string, dueBefore *time.Time) (*ListTicketsResponse, error) {
	url := c.baseURL + "/tickets/" + status
	params := []string{}
	if query != "" {
		params = append(params, "query="+query)
	}
	if dueBefore != nil {
		params = append(params, "due_before="+dueBefore.Format(time.RFC3339))
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
func (c *Client) CreateTicket(title, body, ticketType, repo string, dueDate *time.Time, references []string) (*TicketResponse, error) {
	reqBody := map[string]any{"title": title, "body": body}
	if ticketType != "" {
		reqBody["type"] = ticketType
	}
	if repo != "" {
		reqBody["repo"] = repo
	}
	if dueDate != nil {
		reqBody["due_date"] = dueDate.Format(time.RFC3339)
	}
	if references != nil {
		reqBody["references"] = references
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

// ListArchitectsResponse is the response from GET /architects.
type ListArchitectsResponse struct {
	Architects []ArchitectResponse `json:"projects"`
}

// ListArchitects returns all registered architects from the daemon.
func (c *Client) ListArchitects() (*ListArchitectsResponse, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/architects", nil)
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

	var result ListArchitectsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// UnlinkArchitect removes an architect from the global registry.
// This does not delete any files, only removes the architect from tracking.
func (c *Client) UnlinkArchitect(architectPath string) error {
	req, err := http.NewRequest(http.MethodDelete, c.baseURL+"/architects?path="+architectPath, nil)
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

// UpdateTicket updates a ticket's title, body, and/or references by ID (status-agnostic).
func (c *Client) UpdateTicket(id string, title, body *string, references *[]string) (*TicketResponse, error) {
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
	ArchitectPath string `json:"architect_path"`
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
	if c.architectPath != "" {
		req.Header.Set(ArchitectHeader, c.architectPath)
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

// ListConclusionsParams controls filtering and pagination for ListConclusions.
type ListConclusionsParams struct {
	Type   string
	Limit  int
	Offset int
}

// ListConclusions returns persistent conclusions for the project with optional filtering and pagination.
func (c *Client) ListConclusions(params ListConclusionsParams) (*ListConclusionsResponse, error) {
	url := c.baseURL + "/conclusions"
	var queryParams []string
	if params.Type != "" {
		queryParams = append(queryParams, "type="+params.Type)
	}
	if params.Limit > 0 {
		queryParams = append(queryParams, fmt.Sprintf("limit=%d", params.Limit))
	}
	if params.Offset > 0 {
		queryParams = append(queryParams, fmt.Sprintf("offset=%d", params.Offset))
	}
	if len(queryParams) > 0 {
		url += "?" + strings.Join(queryParams, "&")
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

	var result ListConclusionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetConclusion returns a single conclusion by ID.
func (c *Client) GetConclusion(id string) (*ConclusionResponse, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/conclusions/"+id, nil)
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

	var result ConclusionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// CreateConclusion creates a new conclusion.
func (c *Client) CreateConclusion(conclusionType, ticket, repo, body string) (*ConclusionResponse, error) {
	reqBody := map[string]string{
		"type":   conclusionType,
		"ticket": ticket,
		"repo":   repo,
		"body":   body,
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/conclusions", bytes.NewReader(jsonBody))
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

	var result ConclusionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ListPrompts returns all prompt files with ejection status.
func (c *Client) ListPrompts() (*ListPromptsResponse, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/prompts", nil)
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

	var result ListPromptsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// EjectPrompt ejects a prompt file from base to project for customization.
func (c *Client) EjectPrompt(path string) (*PromptFileInfo, error) {
	reqBody := map[string]string{"path": path}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/prompts/eject", bytes.NewReader(jsonBody))
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

	var result PromptFileInfo
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ResetPrompt resets an ejected prompt to the built-in default by deleting the ejected file.
func (c *Client) ResetPrompt(path string) error {
	reqBody := map[string]string{"path": path}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/prompts/reset", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

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

// EditPromptInEditor opens an ejected prompt in $EDITOR via tmux popup.
func (c *Client) EditPromptInEditor(path string) error {
	reqBody := map[string]string{"path": path}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/prompts/edit", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

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

// EditProjectConfigInEditor opens cortex.yaml in $EDITOR via tmux popup.
func (c *Client) EditProjectConfigInEditor() error {
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/config/project/edit", nil)
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

// RegisterArchitect registers an architect with the daemon.
func (c *Client) RegisterArchitect(path, title string) error {
	reqBody := map[string]string{"path": path}
	if title != "" {
		reqBody["title"] = title
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/architects", bytes.NewReader(jsonBody))
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
