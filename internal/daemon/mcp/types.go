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
	Type     SessionType
	TicketID string // Only set for ticket sessions
}

// Input types for architect tools

// ListTicketsInput is the input for the listTickets tool.
type ListTicketsInput struct {
	Status      string `json:"status" jsonschema:"Ticket status to filter by (required). Must be one of: backlog, progress, review, done"`
	Query       string `json:"query,omitempty" jsonschema:"Optional search term to filter tickets by title/body (case-insensitive substring match)."`
	DueBefore   string `json:"due_before,omitempty" jsonschema:"Optional RFC3339 timestamp to filter tickets with due date before this time."`
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
	Type        string   `json:"type,omitempty" jsonschema:"The ticket type. Available types: 'work' (default implementation), 'debug' (root cause analysis), 'research' (read-only exploration), 'chore' (quick maintenance). Defaults to 'work' if not specified."`
	DueDate     string   `json:"due_date,omitempty" jsonschema:"Optional due date in RFC3339 format (e.g., '2024-12-31T23:59:59Z')."`
	References  []string `json:"references,omitempty" jsonschema:"Cross-references (e.g., 'doc:abc123', 'ticket:xyz789')"`
	ProjectPath string   `json:"project_path,omitempty" jsonschema:"Optional absolute path to target a different registered project. If omitted, uses the current session's project."`
}

// UpdateTicketInput is the input for the updateTicket tool.
type UpdateTicketInput struct {
	ID          string    `json:"id" jsonschema:"The ticket ID to update"`
	Title       *string   `json:"title,omitempty" jsonschema:"New title (optional)"`
	Body        *string   `json:"body,omitempty" jsonschema:"New body (optional)"`
	References  *[]string `json:"references,omitempty" jsonschema:"New references (optional, full replacement)"`
	ProjectPath string    `json:"project_path,omitempty" jsonschema:"Optional absolute path to target a different registered project. If omitted, uses the current session's project."`
}

// DeleteTicketInput is the input for the deleteTicket tool.
type DeleteTicketInput struct {
	ID string `json:"id" jsonschema:"The ticket ID to delete"`
}

// MoveTicketInput is the input for the moveTicket tool.
type MoveTicketInput struct {
	ID          string `json:"id" jsonschema:"The ticket ID to move"`
	Status      string `json:"status" jsonschema:"Target status (backlog/progress/review/done)"`
	ProjectPath string `json:"project_path,omitempty" jsonschema:"Optional absolute path to target a different registered project. If omitted, uses the current session's project."`
}

// SpawnSessionInput is the input for the spawnSession tool.
type SpawnSessionInput struct {
	TicketID    string `json:"ticket_id" jsonschema:"The ticket ID to spawn a session for"`
	Mode        string `json:"mode,omitempty" jsonschema:"Spawn mode: 'normal' (default), 'resume', or 'fresh'"`
	ProjectPath string `json:"project_path,omitempty" jsonschema:"Optional absolute path to target a different registered project. If omitted, uses the current session's project."`
}

// ArchitectAddCommentInput is the input for the architect's addTicketComment tool.
type ArchitectAddCommentInput struct {
	ID          string `json:"id" jsonschema:"The ticket ID to add a comment to"`
	Type        string `json:"type" jsonschema:"Comment type (review_requested/done/blocker/comment)"`
	Content     string `json:"content" jsonschema:"The comment content"`
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

// GetCortexConfigDocsInput is the input for the getCortexConfigDocs tool.
// This tool takes no parameters.
type GetCortexConfigDocsInput struct{}

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

// Input types for ticket tools

// AddCommentInput is the input for the addComment tool.
type AddCommentInput struct {
	Content string `json:"content" jsonschema:"The comment content"`
}

// AddBlockerInput is the input for the addBlocker tool.
type AddBlockerInput struct {
	Content string `json:"content" jsonschema:"Description of the blocker"`
}

// RequestReviewInput is the input for the requestReview tool.
type RequestReviewInput struct {
	RepoPath string `json:"repo_path" jsonschema:"Path to the repository being reviewed"`
	Content  string `json:"content" jsonschema:"Full description of changes for the reviewer"`
	Commit   string `json:"commit,omitempty" jsonschema:"Optional commit hash to review"`
}

// ConcludeSessionInput is the input for the concludeSession tool.
type ConcludeSessionInput struct {
	Content string `json:"content" jsonschema:"Complete summary of work done, decisions made, and files changed"`
}

// CommentOutput is a comment in MCP output (alias to shared type).

type CommentOutput = types.CommentResponse

// MCP-specific output types (structurally different from shared types)

// TicketSummary is a minimal ticket representation for list views.
// Contains only ID and title - use readTicket for full details.
type TicketSummary struct {
	ID    string `json:"id"`
	Title string `json:"title"`
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
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Title      string          `json:"title"`
	Body       string          `json:"body"`
	Tags       []string        `json:"tags,omitempty"`
	References []string        `json:"references,omitempty"`
	Status     string          `json:"status"`
	Created    time.Time       `json:"created"`
	Updated    time.Time       `json:"updated"`
	Due        *time.Time      `json:"due,omitempty"`
	Comments   []CommentOutput `json:"comments"`
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

// AddCommentOutput is the output for the addTicketComment tool.
type AddCommentOutput struct {
	Success bool          `json:"success"`
	Comment CommentOutput `json:"comment,omitempty"`
}

// RequestReviewOutput is the output for the requestReview tool.
type RequestReviewOutput struct {
	Success bool          `json:"success"`
	Message string        `json:"message"`
	Comment CommentOutput `json:"comment"`
}

// ConcludeSessionOutput is the output for the concludeSession tool.
type ConcludeSessionOutput struct {
	Success  bool   `json:"success"`
	TicketID string `json:"ticket_id"`
	Message  string `json:"message,omitempty"`
}

// GetCortexConfigDocsOutput is the output for the getCortexConfigDocs tool.
type GetCortexConfigDocsOutput struct {
	Content    string `json:"content"`
	ConfigName string `json:"config_name"`
}

// UpdateDueDateOutput is the output for the updateDueDate tool.
type UpdateDueDateOutput struct {
	Ticket TicketOutput `json:"ticket"`
}

// ClearDueDateOutput is the output for the clearDueDate tool.
type ClearDueDateOutput struct {
	Ticket TicketOutput `json:"ticket"`
}

// Doc input types

// CreateDocInput is the input for the createDoc tool.
type CreateDocInput struct {
	Title       string   `json:"title" jsonschema:"The document title (required)"`
	Category    string   `json:"category" jsonschema:"Subdirectory/category name (required, e.g., 'specs', 'decisions', 'findings')"`
	Body        string   `json:"body,omitempty" jsonschema:"Markdown body content"`
	Tags        []string `json:"tags,omitempty" jsonschema:"Free-form tags for categorization"`
	References  []string `json:"references,omitempty" jsonschema:"Cross-references (e.g., 'ticket:abc123', 'doc:xyz789')"`
	ProjectPath string   `json:"project_path,omitempty" jsonschema:"Optional absolute path to target a different registered project."`
}

// ReadDocInput is the input for the readDoc tool.
type ReadDocInput struct {
	ID          string `json:"id" jsonschema:"The document ID to read"`
	ProjectPath string `json:"project_path,omitempty" jsonschema:"Optional absolute path to target a different registered project."`
}

// UpdateDocInput is the input for the updateDoc tool.
type UpdateDocInput struct {
	ID          string    `json:"id" jsonschema:"The document ID to update"`
	Title       *string   `json:"title,omitempty" jsonschema:"New title (optional, re-slugs filename)"`
	Body        *string   `json:"body,omitempty" jsonschema:"New body (optional, full replacement)"`
	Tags        *[]string `json:"tags,omitempty" jsonschema:"New tags (optional, full replacement)"`
	References  *[]string `json:"references,omitempty" jsonschema:"New references (optional, full replacement)"`
	ProjectPath string    `json:"project_path,omitempty" jsonschema:"Optional absolute path to target a different registered project."`
}

// DeleteDocInput is the input for the deleteDoc tool.
type DeleteDocInput struct {
	ID string `json:"id" jsonschema:"The document ID to delete"`
}

// MoveDocInput is the input for the moveDoc tool.
type MoveDocInput struct {
	ID          string `json:"id" jsonschema:"The document ID to move"`
	Category    string `json:"category" jsonschema:"Target category/subdirectory"`
	ProjectPath string `json:"project_path,omitempty" jsonschema:"Optional absolute path to target a different registered project."`
}

// ListDocsInput is the input for the listDocs tool.
type ListDocsInput struct {
	Category    string `json:"category,omitempty" jsonschema:"Filter by category/subdirectory"`
	Tag         string `json:"tag,omitempty" jsonschema:"Filter by tag"`
	Query       string `json:"query,omitempty" jsonschema:"Search title and body content (case-insensitive)"`
	ProjectPath string `json:"project_path,omitempty" jsonschema:"Optional absolute path to target a different registered project."`
}

// Doc output types

// DocOutput is the full document representation for MCP.
type DocOutput struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Category   string   `json:"category"`
	Tags       []string `json:"tags"`
	References []string `json:"references"`
	Body       string   `json:"body"`
	Created    string   `json:"created"`
	Updated    string   `json:"updated"`
}

// DocSummaryOutput is a brief view of a doc for list views.
type DocSummaryOutput struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
	Snippet  string   `json:"snippet,omitempty"`
	Created  string   `json:"created"`
	Updated  string   `json:"updated"`
}

// CreateDocOutput is the output for the createDoc tool.
type CreateDocOutput struct {
	Doc DocOutput `json:"doc"`
}

// ReadDocOutput is the output for the readDoc tool.
type ReadDocOutput struct {
	Doc DocOutput `json:"doc"`
}

// UpdateDocOutput is the output for the updateDoc tool.
type UpdateDocOutput struct {
	Doc DocOutput `json:"doc"`
}

// DeleteDocOutput is the output for the deleteDoc tool.
type DeleteDocOutput struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
}

// MoveDocOutput is the output for the moveDoc tool.
type MoveDocOutput struct {
	Doc DocOutput `json:"doc"`
}

// ListDocsOutput is the output for the listDocs tool.
type ListDocsOutput struct {
	Docs  []DocSummaryOutput `json:"docs"`
	Total int                `json:"total"`
}

// Conversion functions

// docResponseToOutput converts an SDK DocResponse to an MCP DocOutput.
func docResponseToOutput(r *types.DocResponse) DocOutput {
	tags := r.Tags
	if tags == nil {
		tags = []string{}
	}
	refs := r.References
	if refs == nil {
		refs = []string{}
	}
	return DocOutput{
		ID:         r.ID,
		Title:      r.Title,
		Category:   r.Category,
		Tags:       tags,
		References: refs,
		Body:       r.Body,
		Created:    r.Created,
		Updated:    r.Updated,
	}
}

// docSummaryToOutput converts an SDK DocSummary to an MCP DocSummaryOutput.
func docSummaryToOutput(s *types.DocSummary) DocSummaryOutput {
	tags := s.Tags
	if tags == nil {
		tags = []string{}
	}
	return DocSummaryOutput{
		ID:       s.ID,
		Title:    s.Title,
		Category: s.Category,
		Tags:     tags,
		Snippet:  s.Snippet,
		Created:  s.Created,
		Updated:  s.Updated,
	}
}

// ticketSummaryResponseToMCP maps a shared TicketSummary (from the HTTP API)
// to the MCP-specific TicketSummary (minimal: only ID and title).
func ticketSummaryResponseToMCP(s *types.TicketSummary) TicketSummary {
	return TicketSummary{
		ID:    s.ID,
		Title: s.Title,
	}
}
