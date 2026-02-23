package conclusion

import (
	"time"

	"github.com/kareemaly/cortex/internal/storage"
)

// ConclusionType represents the type of conclusion.
type ConclusionType string

const (
	TypeArchitect ConclusionType = "architect"
	TypeWork      ConclusionType = "work"
	TypeResearch  ConclusionType = "research"
)

// Re-export shared types from storage.
type (
	NotFoundError   = storage.NotFoundError
	ValidationError = storage.ValidationError
)

// IsNotFound returns true if err is a NotFoundError.
var IsNotFound = storage.IsNotFound

// ConclusionMeta holds the YAML frontmatter fields for a conclusion.
type ConclusionMeta struct {
	ID      string         `yaml:"id"`
	Type    ConclusionType `yaml:"type"`
	Ticket  string         `yaml:"ticket,omitempty"`
	Repo    string         `yaml:"repo,omitempty"`
	Created time.Time      `yaml:"created"`
}

// Conclusion represents a persistent session conclusion with metadata and body.
type Conclusion struct {
	ConclusionMeta
	Body string
}
