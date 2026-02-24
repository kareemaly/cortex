package sdk

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	daemonconfig "github.com/kareemaly/cortex/internal/daemon/config"
	"github.com/kareemaly/cortex/internal/types"
)

const ArchitectHeader = "X-Cortex-Architect"

type Client struct {
	baseURL       string
	httpClient    *http.Client
	architectPath string
}

func NewClient(baseURL, architectPath string) *Client {
	return &Client{
		baseURL:       baseURL,
		architectPath: architectPath,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func DefaultClient(architectPath string) *Client {
	baseURL := os.Getenv("CORTEX_DAEMON_URL")
	if baseURL == "" {
		baseURL = daemonconfig.DefaultDaemonURL
	}
	return NewClient(baseURL, architectPath)
}

func (c *Client) doRequest(req *http.Request) (*http.Response, error) {
	if c.architectPath != "" {
		req.Header.Set(ArchitectHeader, c.architectPath)
	}
	return c.httpClient.Do(req)
}

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
	SpawnCollabResponse      = types.SpawnCollabResponse
)

type APIError struct {
	Code    string
	Message string
	Status  int
}

func (e *APIError) Error() string {
	return e.Message
}

func (e *APIError) IsOrphanedSession() bool {
	return e.Code == "session_orphaned"
}

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

func hasPrefix(id, prefix string) bool {
	return len(prefix) > 0 && len(id) >= len(prefix) && id[:len(prefix)] == prefix
}
