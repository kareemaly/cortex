package ticket

import (
	"time"

	"github.com/kareemaly/cortex/internal/storage"
)

type Status string

const (
	StatusBacklog  Status = "backlog"
	StatusProgress Status = "progress"
	StatusDone     Status = "done"
)

const DefaultTicketType = "work"

type (
	NotFoundError   = storage.NotFoundError
	ValidationError = storage.ValidationError
)

var IsNotFound = storage.IsNotFound

type TicketMeta struct {
	Title      string     `yaml:"title"`
	Repo       string     `yaml:"repo,omitempty"`
	References []string   `yaml:"references,omitempty"`
	Due        *time.Time `yaml:"due,omitempty"`
	Created    time.Time  `yaml:"created"`
	Updated    time.Time  `yaml:"updated"`
}

type Ticket struct {
	ID     string
	Status Status
	TicketMeta
	Body string
}
