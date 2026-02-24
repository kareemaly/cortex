package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

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

// TicketTitle is reused for non-ticket sessions (collab sessions show "Collab: {prompt}")

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
