package prompt

import "fmt"

// NotFoundError indicates a prompt file was not found.
type NotFoundError struct {
	Path string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("prompt file not found: %s\nRun 'cortex install --project' to create default prompts", e.Path)
}
