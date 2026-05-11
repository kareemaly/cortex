package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// SpawnCollabSession spawns a collab session via POST /collab/spawn.
func (c *Client) SpawnCollabSession(path, prompt, slug, mode, variant string) (*SpawnCollabResponse, error) {
	reqBody := map[string]string{"path": path, "prompt": prompt, "slug": slug}
	if mode != "" {
		reqBody["mode"] = mode
	}
	if variant != "" {
		reqBody["variant"] = variant
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/collab/spawn", bytes.NewReader(jsonBody))
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

	var result SpawnCollabResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ConcludeCollabSessionParams holds parameters for concluding a collab session.
type ConcludeCollabSessionParams struct {
	CollabID  string
	Body      string
	StartedAt string
	Commits   []string
}

// ConcludeCollabSession concludes a collab session via POST /collab/{id}/conclude.
func (c *Client) ConcludeCollabSession(p ConcludeCollabSessionParams) (*ConcludeSessionResponse, error) {
	reqBody := map[string]interface{}{"content": p.Body}
	if p.StartedAt != "" {
		reqBody["started_at"] = p.StartedAt
	}
	if len(p.Commits) > 0 {
		reqBody["commits"] = p.Commits
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/collab/"+p.CollabID+"/conclude", bytes.NewReader(jsonBody))
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

// FocusCollabSession focuses the tmux window of a collab session.
func (c *Client) FocusCollabSession(id string) error {
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/collab/"+id+"/focus", nil)
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
