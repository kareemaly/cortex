package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// ConcludeSession concludes a ticket session.
func (c *Client) ConcludeSession(ticketID, content, startedAt string) (*ConcludeSessionResponse, error) {
	reqBody := map[string]string{"content": content}
	if startedAt != "" {
		reqBody["started_at"] = startedAt
	}
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

// EditConclusion opens the conclusion's index.md in $EDITOR via tmux popup.
func (c *Client) EditConclusion(conclusionID string) error {
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/conclusions/"+conclusionID+"/edit", nil)
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
