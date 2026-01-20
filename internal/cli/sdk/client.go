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

// Client is an HTTP client for communicating with the cortex daemon.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new client with the specified base URL.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// DefaultClient returns a client configured for the default daemon address.
func DefaultClient() *Client {
	return NewClient(defaultBaseURL)
}

// HealthResponse is the response from the health endpoint.
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// TicketSummary is a brief view of a ticket.
type TicketSummary struct {
	ID                string    `json:"id"`
	Title             string    `json:"title"`
	Status            string    `json:"status"`
	Created           time.Time `json:"created"`
	HasActiveSessions bool      `json:"has_active_sessions"`
}

// ListAllTicketsResponse groups tickets by status.
type ListAllTicketsResponse struct {
	Backlog  []TicketSummary `json:"backlog"`
	Progress []TicketSummary `json:"progress"`
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
	Approved *time.Time `json:"approved"`
}

// ReportResponse is the report portion of a session response.
type ReportResponse struct {
	Files        []string `json:"files"`
	ScopeChanges *string  `json:"scope_changes,omitempty"`
	Decisions    []string `json:"decisions"`
	Summary      string   `json:"summary"`
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
	GitBase       map[string]string     `json:"git_base"`
	Report        ReportResponse        `json:"report"`
	CurrentStatus *StatusEntryResponse  `json:"current_status,omitempty"`
	StatusHistory []StatusEntryResponse `json:"status_history"`
}

// TicketResponse is the full ticket response with status.
type TicketResponse struct {
	ID       string            `json:"id"`
	Title    string            `json:"title"`
	Body     string            `json:"body"`
	Status   string            `json:"status"`
	Dates    DatesResponse     `json:"dates"`
	Sessions []SessionResponse `json:"sessions"`
}

// ErrorResponse is the standard error response format.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Details string `json:"details,omitempty"`
}

// Health checks if the daemon is healthy.
func (c *Client) Health() error {
	resp, err := c.httpClient.Get(c.baseURL + "/health")
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
	resp, err := c.httpClient.Get(c.baseURL + "/health")
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
func (c *Client) ListAllTickets() (*ListAllTicketsResponse, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/tickets")
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
func (c *Client) ListTicketsByStatus(status string) (*ListTicketsResponse, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/tickets/" + status)
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
	resp, err := c.httpClient.Get(c.baseURL + "/tickets/" + status + "/" + id)
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

	resp, err := c.httpClient.Post(c.baseURL+"/tickets", "application/json", bytes.NewReader(jsonBody))
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

// SpawnSession spawns a new session for a ticket.
func (c *Client) SpawnSession(status, id string) (*SessionResponse, error) {
	resp, err := c.httpClient.Post(c.baseURL+"/tickets/"+status+"/"+id+"/spawn", "application/json", nil)
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

	resp, err := c.httpClient.Do(req)
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
	all, err := c.ListAllTickets()
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
	for _, summary := range all.Done {
		if summary.ID == ticketID || hasPrefix(summary.ID, ticketID) {
			return c.GetTicket("done", summary.ID)
		}
	}

	return nil, fmt.Errorf("ticket not found: %s", ticketID)
}

// FindSession searches for a session by ID across all tickets.
func (c *Client) FindSession(sessionID string) (*SessionResponse, *TicketResponse, error) {
	all, err := c.ListAllTickets()
	if err != nil {
		return nil, nil, err
	}

	// Helper to search in a status
	searchStatus := func(summaries []TicketSummary, status string) (*SessionResponse, *TicketResponse, error) {
		for _, summary := range summaries {
			ticket, err := c.GetTicket(status, summary.ID)
			if err != nil {
				continue
			}
			for i := range ticket.Sessions {
				s := &ticket.Sessions[i]
				if s.ID == sessionID || hasPrefix(s.ID, sessionID) {
					return s, ticket, nil
				}
			}
		}
		return nil, nil, nil
	}

	// Search backlog
	if session, ticket, err := searchStatus(all.Backlog, "backlog"); err != nil {
		return nil, nil, err
	} else if session != nil {
		return session, ticket, nil
	}

	// Search progress
	if session, ticket, err := searchStatus(all.Progress, "progress"); err != nil {
		return nil, nil, err
	} else if session != nil {
		return session, ticket, nil
	}

	// Search done
	if session, ticket, err := searchStatus(all.Done, "done"); err != nil {
		return nil, nil, err
	} else if session != nil {
		return session, ticket, nil
	}

	return nil, nil, fmt.Errorf("session not found: %s", sessionID)
}

// parseError extracts an error message from a non-OK response.
func (c *Client) parseError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	if errResp.Details != "" {
		return fmt.Errorf("%s", errResp.Details)
	}
	if errResp.Error != "" {
		return fmt.Errorf("%s", errResp.Error)
	}
	return fmt.Errorf("request failed with status %d", resp.StatusCode)
}

// hasPrefix checks if id starts with prefix (for short ID matching).
func hasPrefix(id, prefix string) bool {
	return len(prefix) > 0 && len(id) >= len(prefix) && id[:len(prefix)] == prefix
}
