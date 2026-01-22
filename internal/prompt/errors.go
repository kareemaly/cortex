package prompt

import "fmt"

// NotFoundError indicates a prompt template file was not found.
type NotFoundError struct {
	Path string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("prompt template not found: %s\nRun 'cortex install --project' to create default templates", e.Path)
}

// ParseError indicates a template parsing failure.
type ParseError struct {
	Path string
	Err  error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("failed to parse template %s: %v", e.Path, e.Err)
}

func (e *ParseError) Unwrap() error {
	return e.Err
}

// RenderError indicates a template rendering failure.
type RenderError struct {
	Path string
	Err  error
}

func (e *RenderError) Error() string {
	return fmt.Sprintf("failed to render template %s: %v", e.Path, e.Err)
}

func (e *RenderError) Unwrap() error {
	return e.Err
}
