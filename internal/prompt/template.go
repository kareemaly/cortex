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
	Comments       string // pre-formatted comments block
	References     string // pre-formatted references block
	IsWorktree     bool   // true when running in a worktree
	WorktreePath   string // worktree only
	WorktreeBranch string // worktree only
}

// ArchitectKickoffVars contains variables for the architect kickoff template.
type ArchitectKickoffVars struct {
	ProjectName string
	TicketList  string
	CurrentDate string
	TopTags     string // comma-separated top tags
	DocsList    string // formatted recent docs list
}

// MetaKickoffVars contains variables for the meta kickoff template.
type MetaKickoffVars struct {
	CurrentDate string
	ProjectList string
	SessionList string
}

// RenderTemplate renders a template string with the given variables.
func RenderTemplate(content string, vars any) (string, error) {
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
