package sdk

import (
	"fmt"
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
