package mcp

import (
	"time"

	"github.com/kareemaly/cortex/internal/types"
)

// SessionType indicates the type of MCP session.
type SessionType string

const (
	SessionTypeArchitect SessionType = "architect"
	SessionTypeTicket    SessionType = "ticket"
	SessionTypeCollab    SessionType = "collab"
)

// Session holds the current session context.
type Session struct {
	Type       SessionType
	TicketID   string // Only set for ticket sessions
	TicketType string // Only set for ticket sessions
	CollabID   string // Only set for collab sessions
	Repo       string // from CORTEX_REPO env var
}

// SpawnCollabSessionInput is the input for the spawnCollabSession tool.
type SpawnCollabSessionInput struct {
	Path    string `json:"path" jsonschema:"Any valid filesystem path where the collab agent will spawn (required). The path must exist on disk."`
	Prompt  string `json:"prompt" jsonschema:"Brief question or topic to discuss. Keep it minimal — the collab agent starts in the directory with its own AGENTS.md context."`
	Variant string `json:"variant" jsonschema:"Agent variant name from the agents map in cortex.yaml (required). Use listVariants to see available names."`
}

// SpawnCollabSessionOutput is the output for the spawnCollabSession tool.
type SpawnCollabSessionOutput struct {
	Success    bool   `json:"success"`
	CollabID   string `json:"collab_id,omitempty"`
	TmuxWindow string `json:"tmux_window,omitempty"`
	State      string `json:"state,omitempty"`
	Message    string `json:"message,omitempty"`
}

// CollabConcludeOutput is the output for the collab concludeSession tool.
type CollabConcludeOutput struct {
	Success  bool   `json:"success"`
	CollabID string `json:"collab_id"`
	Message  string `json:"message,omitempty"`
}

// Input types for architect tools

// ListTicketsInput is the input for the listTickets tool.
type ListTicketsInput struct {
	Status string `json:"status" jsonschema:"Ticket status to filter by (required). Must be one of: backlog, progress, done"`
	Query  string `json:"query,omitempty" jsonschema:"Optional search term to filter tickets by title/body (case-insensitive substring match)."`
}

// ReadTicketInput is the input for the readTicket tool.
type ReadTicketInput struct {
	ID string `json:"id" jsonschema:"The ticket ID to read"`
}

// CreateWorkTicketInput is the input for the createWorkTicket tool.
type CreateWorkTicketInput struct {
	Title      string   `json:"title" jsonschema:"The ticket title (required)"`
	Body       string   `json:"body,omitempty" jsonschema:"The ticket body/description"`
	Repo       string   `json:"repo" jsonschema:"Repository path for this ticket (required). Must be from the configured repos list in cortex.yaml."`
	DueDate    string   `json:"due_date,omitempty" jsonschema:"Optional due date in RFC3339 format (e.g., '2024-12-31T23:59:59Z')."`
	References []string `json:"references,omitempty" jsonschema:"Ticket IDs to reference (plain ticket IDs only, no prefix scheme)"`
}

// CreateFollowUpTicketInput is the input for the createFollowUpTicket tool.
type CreateFollowUpTicketInput struct {
	Title   string `json:"title" jsonschema:"The follow-up ticket title (required)"`
	Body    string `json:"body,omitempty" jsonschema:"The ticket body/description"`
	Repo    string `json:"repo" jsonschema:"Repository path for this ticket (required). Must be in the architect's repos whitelist."`
	DueDate string `json:"due_date,omitempty" jsonschema:"Optional due date in RFC3339 format (e.g., '2024-12-31T23:59:59Z')"`
}

// CreateFollowUpTicketOutput is the output for the createFollowUpTicket tool.
type CreateFollowUpTicketOutput struct {
	Ticket TicketOutput `json:"ticket"`
}

// UpdateTicketInput is the input for the updateTicket tool.
type UpdateTicketInput struct {
	ID         string    `json:"id" jsonschema:"The ticket ID to update"`
	Title      *string   `json:"title,omitempty" jsonschema:"New title (optional)"`
	Body       *string   `json:"body,omitempty" jsonschema:"New body (optional)"`
	References *[]string `json:"references,omitempty" jsonschema:"Ticket IDs to reference (optional, full replacement — plain ticket IDs only, no prefix scheme)"`
}

// DeleteTicketInput is the input for the deleteTicket tool.
type DeleteTicketInput struct {
	ID string `json:"id" jsonschema:"The ticket ID to delete"`
}

// MoveTicketInput is the input for the moveTicket tool.
type MoveTicketInput struct {
	ID     string `json:"id" jsonschema:"The ticket ID to move"`
	Status string `json:"status" jsonschema:"Target status (backlog/progress/done)"`
}

// SpawnSessionInput is the input for the spawnSession tool.
type SpawnSessionInput struct {
	TicketID string `json:"ticket_id" jsonschema:"The ticket ID to spawn a session for"`
	Mode     string `json:"mode,omitempty" jsonschema:"Spawn mode: 'normal' (default), 'resume', or 'fresh'"`
	Variant  string `json:"variant" jsonschema:"Agent variant name from the agents map in cortex.yaml (required). Use listVariants to see available names."`
}

// ListVariantsInput is the input for the listVariants tool (no parameters needed).
type ListVariantsInput struct{}

// ListVariantsOutput is the output for the listVariants tool.
type ListVariantsOutput struct {
	Variants []string `json:"variants"`
}

// UpdateDueDateInput is the input for the updateDueDate tool.
type UpdateDueDateInput struct {
	ID      string `json:"id" jsonschema:"The ticket ID (required)"`
	DueDate string `json:"due_date" jsonschema:"The due date in RFC3339 format (required, e.g., '2024-12-31T23:59:59Z')"`
}

// ClearDueDateInput is the input for the clearDueDate tool.
type ClearDueDateInput struct {
	ID string `json:"id" jsonschema:"The ticket ID (required)"`
}

// ConcludeSessionInput is the input for the concludeSession tool.
type ConcludeSessionInput struct {
	Body            string   `json:"body" jsonschema:"Session conclusion summary — outcome, files changed, and follow-up work or blockers (required)."`
	Commits         []string `json:"commits,omitempty" jsonschema:"List of commit SHAs produced during this session. Required for work ticket sessions unless rejected=true. Optional for collab sessions. Ignored for architect sessions."`
	Rejected        bool     `json:"rejected,omitempty" jsonschema:"Set to true if the session produced no work and should be marked as rejected. Requires rejection_reason. Work ticket sessions only."`
	RejectionReason string   `json:"rejection_reason,omitempty" jsonschema:"Required and non-empty when rejected=true. Explain why the session produced no commits. Work ticket sessions only."`
}

// MCP-specific output types (structurally different from shared types)

// TicketSummary is an enriched ticket representation for list views.
type TicketSummary struct {
	ID      string     `json:"id"`
	Title   string     `json:"title"`
	Type    string     `json:"type"`
	Repo    string     `json:"repo,omitempty"`
	Path    string     `json:"path,omitempty"`
	Due     *time.Time `json:"due,omitempty"`
	Created time.Time  `json:"created"`
	Updated time.Time  `json:"updated"`
}

// SessionOutput represents a work session.
type SessionOutput struct {
	Agent      string  `json:"agent"`
	TmuxWindow string  `json:"tmux_window"`
	Status     string  `json:"status"`
	Tool       *string `json:"tool,omitempty"`
}

// TicketOutput is the full ticket representation.
type TicketOutput struct {
	ID           string            `json:"id"`
	Type         string            `json:"type"`
	Title        string            `json:"title"`
	Body         string            `json:"body"`
	Repo         string            `json:"repo,omitempty"`
	Path         string            `json:"path,omitempty"`
	ConclusionID string            `json:"conclusion_id,omitempty"`
	References   []string          `json:"references,omitempty"`
	Status       string            `json:"status"`
	Created      time.Time         `json:"created"`
	Updated      time.Time         `json:"updated"`
	Due          *time.Time        `json:"due,omitempty"`
	Conclusion   *ConclusionOutput `json:"conclusion,omitempty"`
}

// Tool output wrappers

// ListTicketsOutput is the output for the listTickets tool.
type ListTicketsOutput struct {
	Tickets []TicketSummary `json:"tickets"`
	Total   int             `json:"total"`
}

// ReadTicketOutput is the output for the readTicket tool.
type ReadTicketOutput struct {
	Ticket TicketOutput `json:"ticket"`
}

// CreateTicketOutput is the output for the createTicket tool.
type CreateTicketOutput struct {
	Ticket TicketOutput `json:"ticket"`
}

// UpdateTicketOutput is the output for the updateTicket tool.
type UpdateTicketOutput struct {
	Ticket TicketOutput `json:"ticket"`
}

// DeleteTicketOutput is the output for the deleteTicket tool.
type DeleteTicketOutput struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
}

// MoveTicketOutput is the output for the moveTicket tool.
type MoveTicketOutput struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
	Status  string `json:"status"`
}

// SpawnSessionOutput is the output for the spawnSession tool.
type SpawnSessionOutput struct {
	Success    bool   `json:"success"`
	TicketID   string `json:"ticket_id,omitempty"`
	TmuxWindow string `json:"tmux_window,omitempty"`
	State      string `json:"state,omitempty"`
	Message    string `json:"message,omitempty"`
}

// ConcludeSessionOutput is the output for the concludeSession tool.
type ConcludeSessionOutput struct {
	Success  bool   `json:"success"`
	TicketID string `json:"ticket_id"`
	Message  string `json:"message,omitempty"`
}

// UpdateDueDateOutput is the output for the updateDueDate tool.
type UpdateDueDateOutput struct {
	Ticket TicketOutput `json:"ticket"`
}

// ClearDueDateOutput is the output for the clearDueDate tool.
type ClearDueDateOutput struct {
	Ticket TicketOutput `json:"ticket"`
}

// ArchitectConcludeOutput is the output for the architect concludeSession tool.
type ArchitectConcludeOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// Conclusion types

// ListConclusionsInput is the input for the listConclusions tool.
type ListConclusionsInput struct {
	Type   string `json:"type,omitempty" jsonschema:"Filter by type: architect, work, or collab."`
	Limit  int    `json:"limit,omitempty" jsonschema:"Max results to return (default 10)."`
	Offset int    `json:"offset,omitempty" jsonschema:"Results to skip for pagination (default 0)."`
}

// ReadConclusionInput is the input for the readConclusion tool.
type ReadConclusionInput struct {
	ID string `json:"id" jsonschema:"The conclusion ID to read"`
}

// ConclusionListItem is a metadata-only conclusion record for list responses (no body).
type ConclusionListItem struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Ticket      string `json:"ticket,omitempty"`
	Repo        string `json:"repo,omitempty"`
	ConcludedAt string `json:"concluded_at"`
	StartedAt   string `json:"started_at,omitempty"`
}

// ConclusionOutput is a full conclusion record including the body.
type ConclusionOutput struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Ticket      string `json:"ticket,omitempty"`
	Repo        string `json:"repo,omitempty"`
	Body        string `json:"body"`
	ConcludedAt string `json:"concluded_at"`
	StartedAt   string `json:"started_at,omitempty"`
}

// ListConclusionsOutput is the output for the listConclusions tool.
type ListConclusionsOutput struct {
	Conclusions []ConclusionListItem `json:"conclusions"`
	Total       int                  `json:"total"`
}

// ReadConclusionOutput is the output for the readConclusion tool.
type ReadConclusionOutput struct {
	Conclusion ConclusionOutput `json:"conclusion"`
}

// SearchInput is the input for the search tool.
type SearchInput struct {
	Query string `json:"query" jsonschema:"Search term — case-insensitive substring matched against ticket title+body and conclusion body (required)."`
	Limit int    `json:"limit,omitempty" jsonschema:"Max results to return (default 25)."`
}

// SearchResultItem is one entry in search results — either a ticket (with optional nested conclusion) or a bare ticketless conclusion.
type SearchResultItem struct {
	Ticket     *TicketOutput     `json:"ticket,omitempty"`
	Conclusion *ConclusionOutput `json:"conclusion,omitempty"`
}

// SearchOutput is the output for the search tool.
type SearchOutput struct {
	Results []SearchResultItem `json:"results"`
	Total   int                `json:"total"`
}

// ticketSummaryResponseToMCP maps a shared TicketSummary (from the HTTP API)
// to the MCP-specific TicketSummary with enriched fields.
func ticketSummaryResponseToMCP(s *types.TicketSummary) TicketSummary {
	return TicketSummary{
		ID:      s.ID,
		Title:   s.Title,
		Type:    s.Type,
		Due:     s.Due,
		Created: s.Created,
		Updated: s.Updated,
	}
}
