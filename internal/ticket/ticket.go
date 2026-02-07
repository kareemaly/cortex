package ticket

import (
	"time"

	"github.com/kareemaly/cortex/internal/storage"
)

// Status represents a ticket's workflow status.
type Status string

const (
	StatusBacklog  Status = "backlog"
	StatusProgress Status = "progress"
	StatusReview   Status = "review"
	StatusDone     Status = "done"
)

// DefaultTicketType is the default type for tickets when none is specified.
const DefaultTicketType = "work"

// Re-export shared types from storage.
type (
	CommentType     = storage.CommentType
	Comment         = storage.Comment
	CommentAction   = storage.CommentAction
	NotFoundError   = storage.NotFoundError
	ValidationError = storage.ValidationError
)

// Re-export shared comment type constants.
var (
	CommentReviewRequested = storage.CommentReviewRequested
	CommentDone            = storage.CommentDone
	CommentBlocker         = storage.CommentBlocker
	CommentGeneral         = storage.CommentGeneral
)

// IsNotFound returns true if err is a NotFoundError.
var IsNotFound = storage.IsNotFound

// GitDiffArgs holds the arguments for a git_diff action.
type GitDiffArgs = storage.GitDiffArgs

// TicketMeta holds the YAML frontmatter fields for a ticket.
type TicketMeta struct {
	ID         string     `yaml:"id"`
	Title      string     `yaml:"title"`
	Type       string     `yaml:"type"`
	Tags       []string   `yaml:"tags,omitempty"`
	References []string   `yaml:"references,omitempty"`
	Due        *time.Time `yaml:"due,omitempty"`
	Created    time.Time  `yaml:"created"`
	Updated    time.Time  `yaml:"updated"`
}

// Ticket represents a work item with metadata, body, and comments.
type Ticket struct {
	TicketMeta
	Body     string
	Comments []Comment
}
