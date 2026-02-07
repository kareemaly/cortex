package docs

import (
	"time"

	"github.com/kareemaly/cortex/internal/storage"
)

// Re-export shared types from storage.
type (
	Comment         = storage.Comment
	CommentType     = storage.CommentType
	CommentAction   = storage.CommentAction
	NotFoundError   = storage.NotFoundError
	ValidationError = storage.ValidationError
)

// IsNotFound returns true if err is a NotFoundError.
var IsNotFound = storage.IsNotFound

// DocMeta holds the YAML frontmatter fields for a doc.
// Category is NOT in frontmatter — derived from directory path.
type DocMeta struct {
	ID         string    `yaml:"id"`
	Title      string    `yaml:"title"`
	Tags       []string  `yaml:"tags,omitempty"`
	References []string  `yaml:"references,omitempty"`
	Created    time.Time `yaml:"created"`
	Updated    time.Time `yaml:"updated"`
}

// Doc represents a documentation file with metadata, category, body, and comments.
type Doc struct {
	DocMeta
	Category string    // derived from parent directory name at load time
	Body     string    // markdown body
	Comments []Comment // loaded from comment-*.md files
}
