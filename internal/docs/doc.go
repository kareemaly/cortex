package docs

import "time"

// Doc represents a documentation file with YAML frontmatter and markdown body.
type Doc struct {
	ID         string    `yaml:"id"`
	Title      string    `yaml:"title"`
	Category   string    `yaml:"category"`
	Tags       []string  `yaml:"tags,omitempty"`
	References []string  `yaml:"references,omitempty"`
	Created    time.Time `yaml:"created"`
	Updated    time.Time `yaml:"updated"`
	Body       string    `yaml:"-"` // Not in frontmatter; stored as markdown after ---
}
