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
)

// Session holds the current session context.
type Session struct {
	Type       SessionType
	TicketID   string // Only set for ticket sessions
	TicketType string // Only set for ticket sessions
	Repo       string // from CORTEX_REPO env var
}

// Input types for architect tools

// ListTicketsInput is the input for the listTickets tool.
type ListTicketsInput struct {
	Status      string `json:"status" jsonschema:"Ticket status to filter by (required). Must be one of: backlog, progress, done"`
	Query       string `json:"query,omitempty" jsonschema:"Optional search term to filter tickets by title/body (case-insensitive substring match)."`
	DueBefore   string `json:"due_before,omitempty" jsonschema:"Optional RFC3339 timestamp to filter tickets with due date before this time."`
	Tag         string `json:"tag,omitempty" jsonschema:"Optional tag to filter tickets (case-insensitive)."`
	ProjectPath string `json:"project_path,omitempty" jsonschema:"Optional absolute path to target a different registered project. If omitted, uses the current session's project."`
}

// ReadTicketInput is the input for the readTicket tool.
type ReadTicketInput struct {
	ID          string `json:"id" jsonschema:"The ticket ID to read"`
	ProjectPath string `json:"project_path,omitempty" jsonschema:"Optional absolute path to target a different registered project. If omitted, uses the current session's project."`
}

// CreateTicketInput is the input for the createTicket tool.
type CreateTicketInput struct {
	Title       string   `json:"title" jsonschema:"The ticket title (required)"`
	Body        string   `json:"body,omitempty" jsonschema:"The ticket body/description"`
	Type        string   `json:"type,omitempty" jsonschema:"The ticket type. Must match a type defined in the project's cortex.yaml ticket config. Defaults to 'work' if not specified."`
	Repo        string   `json:"repo,omitempty" jsonschema:"Optional repository path for this ticket"`
	DueDate     string   `json:"due_date,omitempty" jsonschema:"Optional due date in RFC3339 format (e.g., '2024-12-31T23:59:59Z')."`
	References  []string `json:"references,omitempty" jsonschema:"Ticket IDs to reference (plain ticket IDs only, no prefix scheme)"`
	Tags        []string `json:"tags,omitempty" jsonschema:"Free-form tags for categorization"`
	ProjectPath string   `json:"project_path,omitempty" jsonschema:"Optional absolute path to target a different registered project. If omitted, uses the current session's project."`
}

// UpdateTicketInput is the input for the updateTicket tool.
type UpdateTicketInput struct {
	ID          string    `json:"id" jsonschema:"The ticket ID to update"`
	Title       *string   `json:"title,omitempty" jsonschema:"New title (optional)"`
	Body        *string   `json:"body,omitempty" jsonschema:"New body (optional)"`
	Type        *string   `json:"type,omitempty" jsonschema:"New ticket type (optional). Must match a type defined in project config."`
	References  *[]string `json:"references,omitempty" jsonschema:"Ticket IDs to reference (optional, full replacement — plain ticket IDs only, no prefix scheme)"`
	Tags        *[]string `json:"tags,omitempty" jsonschema:"New tags (optional, full replacement)"`
	ProjectPath string    `json:"project_path,omitempty" jsonschema:"Optional absolute path to target a different registered project. If omitted, uses the current session's project."`
}

// DeleteTicketInput is the input for the deleteTicket tool.
type DeleteTicketInput struct {
	ID string `json:"id" jsonschema:"The ticket ID to delete"`
}

// MoveTicketInput is the input for the moveTicket tool.
type MoveTicketInput struct {
	ID          string `json:"id" jsonschema:"The ticket ID to move"`
	Status      string `json:"status" jsonschema:"Target status (backlog/progress/done)"`
	ProjectPath string `json:"project_path,omitempty" jsonschema:"Optional absolute path to target a different registered project. If omitted, uses the current session's project."`
}

// SpawnSessionInput is the input for the spawnSession tool.
type SpawnSessionInput struct {
	TicketID    string `json:"ticket_id" jsonschema:"The ticket ID to spawn a session for"`
	Mode        string `json:"mode,omitempty" jsonschema:"Spawn mode: 'normal' (default), 'resume', or 'fresh'"`
	ProjectPath string `json:"project_path,omitempty" jsonschema:"Optional absolute path to target a different registered project. If omitted, uses the current session's project."`
}

// UpdateDueDateInput is the input for the updateDueDate tool.
type UpdateDueDateInput struct {
	ID          string `json:"id" jsonschema:"The ticket ID (required)"`
	DueDate     string `json:"due_date" jsonschema:"The due date in RFC3339 format (required, e.g., '2024-12-31T23:59:59Z')"`
	ProjectPath string `json:"project_path,omitempty" jsonschema:"Optional absolute path to target a different registered project. If omitted, uses the current session's project."`
}

// ClearDueDateInput is the input for the clearDueDate tool.
type ClearDueDateInput struct {
	ID          string `json:"id" jsonschema:"The ticket ID (required)"`
	ProjectPath string `json:"project_path,omitempty" jsonschema:"Optional absolute path to target a different registered project. If omitted, uses the current session's project."`
}

// ListProjectsInput is the input for the listProjects tool.
// This tool takes no parameters.
type ListProjectsInput struct{}

// ProjectSummary represents a project in the listProjects output.
type ProjectSummary struct {
	Path   string `json:"path"`
	Title  string `json:"title"`
	Exists bool   `json:"exists"`
}

// ListProjectsOutput is the output for the listProjects tool.
type ListProjectsOutput struct {
	Projects []ProjectSummary `json:"projects"`
}

// ConcludeSessionInput is the input for the concludeSession tool.
type ConcludeSessionInput struct {
	Content string `json:"content" jsonschema:"Complete summary of work done, decisions made, and files changed"`
}

// MCP-specific output types (structurally different from shared types)

// TicketSummary is an enriched ticket representation for list views.
type TicketSummary struct {
	ID      string     `json:"id"`
	Title   string     `json:"title"`
	Type    string     `json:"type"`
	Repo    string     `json:"repo,omitempty"`
	Tags    []string   `json:"tags,omitempty"`
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
	ID         string     `json:"id"`
	Type       string     `json:"type"`
	Title      string     `json:"title"`
	Body       string     `json:"body"`
	Repo       string     `json:"repo,omitempty"`
	Session    string     `json:"session,omitempty"`
	Tags       []string   `json:"tags,omitempty"`
	References []string   `json:"references,omitempty"`
	Status     string     `json:"status"`
	Created    time.Time  `json:"created"`
	Updated    time.Time  `json:"updated"`
	Due        *time.Time `json:"due,omitempty"`
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
	SessionID  string `json:"session_id,omitempty"`
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

// ListConclusionsInput is the input for the listSessions tool (now querying persistent conclusions).
type ListConclusionsInput struct {
	ProjectPath string `json:"project_path,omitempty" jsonschema:"Optional absolute path to target a different registered project."`
}

// ReadConclusionInput is the input for the readSession tool.
type ReadConclusionInput struct {
	ID          string `json:"id" jsonschema:"The conclusion/session ID to read"`
	ProjectPath string `json:"project_path,omitempty" jsonschema:"Optional absolute path to target a different registered project."`
}

// ConclusionOutput is a persistent session/conclusion record.
type ConclusionOutput struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Ticket  string `json:"ticket,omitempty"`
	Repo    string `json:"repo,omitempty"`
	Body    string `json:"body"`
	Created string `json:"created"`
}

// ListConclusionsOutput is the output for the listSessions tool.
type ListConclusionsOutput struct {
	Conclusions []ConclusionOutput `json:"conclusions"`
	Total       int                `json:"total"`
}

// ReadConclusionOutput is the output for the readSession tool.
type ReadConclusionOutput struct {
	Conclusion ConclusionOutput `json:"conclusion"`
}

// ticketSummaryResponseToMCP maps a shared TicketSummary (from the HTTP API)
// to the MCP-specific TicketSummary with enriched fields.
func ticketSummaryResponseToMCP(s *types.TicketSummary) TicketSummary {
	return TicketSummary{
		ID:      s.ID,
		Title:   s.Title,
		Type:    s.Type,
		Tags:    s.Tags,
		Due:     s.Due,
		Created: s.Created,
		Updated: s.Updated,
	}
}
