package prompt

import (
	"bytes"
	"text/template"
)

// TicketVars contains variables available for ticket prompt templates.
type TicketVars struct {
	ProjectPath    string
	TicketID       string
	TicketTitle    string
	TicketBody     string
	WorktreePath   string // worktree only
	WorktreeBranch string // worktree only
}

// RenderTemplate renders a template string with the given variables.
func RenderTemplate(content string, vars TicketVars) (string, error) {
	tmpl, err := template.New("prompt").Parse(content)
	if err != nil {
		return "", &TemplateError{Message: "failed to parse template", Err: err}
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", &TemplateError{Message: "failed to execute template", Err: err}
	}

	return buf.String(), nil
}

// TemplateError represents a template rendering error.
type TemplateError struct {
	Message string
	Err     error
}

func (e *TemplateError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *TemplateError) Unwrap() error {
	return e.Err
}
