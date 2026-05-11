package prompt

import (
	"bytes"
	"text/template"
)

// TicketVars contains variables available for ticket prompt templates.
type TicketVars struct {
	ProjectPath   string
	TicketID      string
	TicketTitle   string
	TicketBody    string
	References    string // pre-formatted references block
	Repo          string // stable repo key for the ticket
	RepoPath      string // resolved local path for the ticket repo key
	ArchitectName string // architect name from config
	Repos         string // formatted list of other repos in the ecosystem (excluding current repo)
}

// ArchitectKickoffVars contains variables for the architect kickoff template.
type ArchitectKickoffVars struct {
	ArchitectName    string
	TicketList       string
	CurrentDate      string
	Sessions         string // recent conclusions list
	Repos            string // configured repo list
	LastConclusionID string // ID of most recent architect conclusion, empty if none
	Variants         string // comma-separated agent variant names, empty if none configured
}

// TicketsVars contains status-specific ticket lists for architect templates.
type TicketsVars struct {
	Backlog  string
	Progress string
	Done     string
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
