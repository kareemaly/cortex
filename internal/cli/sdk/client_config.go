package sdk

import (
	"fmt"
	"net/http"
)

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
