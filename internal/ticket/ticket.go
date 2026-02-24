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
	StatusDone     Status = "done"
)

// DefaultTicketType is the default type for tickets when none is specified.
const DefaultTicketType = "work"

// Re-export shared types from storage.
type (
	NotFoundError   = storage.NotFoundError
	ValidationError = storage.ValidationError
)

// IsNotFound returns true if err is a NotFoundError.
var IsNotFound = storage.IsNotFound

// TicketMeta holds the YAML frontmatter fields for a ticket.
type TicketMeta struct {
	ID         string     `yaml:"id"`
	Title      string     `yaml:"title"`
	Type       string     `yaml:"type"`
	Repo       string     `yaml:"repo,omitempty"`
	Path       string     `yaml:"path,omitempty"`
	Session    string     `yaml:"session,omitempty"`
	References []string   `yaml:"references,omitempty"`
	Due        *time.Time `yaml:"due,omitempty"`
	Created    time.Time  `yaml:"created"`
	Updated    time.Time  `yaml:"updated"`
}

// Ticket represents a work item with metadata and body.
type Ticket struct {
	TicketMeta
	Body string
}
