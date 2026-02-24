package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

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
func (c *Client) CreateTicket(title, body, ticketType, repo, path string, dueDate *time.Time, references []string) (*TicketResponse, error) {
	reqBody := map[string]any{"title": title, "body": body}
	if ticketType != "" {
		reqBody["type"] = ticketType
	}
	if repo != "" {
		reqBody["repo"] = repo
	}
	if path != "" {
		reqBody["path"] = path
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

// UpdateTicket updates a ticket's title, body, and/or references by ID (status-agnostic).
func (c *Client) UpdateTicket(id string, title, body *string, references *[]string) (*TicketResponse, error) {
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

// EditTicket opens the ticket's index.md in $EDITOR via tmux popup.
func (c *Client) EditTicket(ticketID string) error {
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/tickets/"+ticketID+"/edit", nil)
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
