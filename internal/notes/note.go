package notes

import "time"

// Note represents a lightweight reminder/note.
type Note struct {
	ID      string     `yaml:"id" json:"id"`
	Text    string     `yaml:"text" json:"text"`
	Due     *time.Time `yaml:"due,omitempty" json:"due,omitempty"`
	Created time.Time  `yaml:"created" json:"created"`
}
