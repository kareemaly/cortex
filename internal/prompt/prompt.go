package prompt

import (
	"bytes"
	"os"
	"path/filepath"
	"text/template"
)

// ArchitectVars contains variables for architect prompt templates.
type ArchitectVars struct {
	ProjectName string
	TmuxSession string
}

// TicketVars contains variables for ticket agent prompt templates.
type TicketVars struct {
	TicketID string
	Title    string
	Body     string
	Slug     string
}

// PromptsDir returns the path to the prompts directory.
func PromptsDir(projectRoot string) string {
	return filepath.Join(projectRoot, ".cortex", "prompts")
}

// ArchitectPath returns the path to the architect prompt template.
func ArchitectPath(projectRoot string) string {
	return filepath.Join(PromptsDir(projectRoot), "architect.md")
}

// TicketAgentPath returns the path to the ticket agent prompt template.
func TicketAgentPath(projectRoot string) string {
	return filepath.Join(PromptsDir(projectRoot), "ticket-agent.md")
}

// LoadArchitect loads and renders the architect prompt template.
func LoadArchitect(projectRoot string, vars ArchitectVars) (string, error) {
	path := ArchitectPath(projectRoot)
	return loadAndRender(path, vars)
}

// LoadTicketAgent loads and renders the ticket agent prompt template.
func LoadTicketAgent(projectRoot string, vars TicketVars) (string, error) {
	path := TicketAgentPath(projectRoot)
	return loadAndRender(path, vars)
}

// loadAndRender loads a template file and renders it with the given data.
func loadAndRender(path string, data any) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", &NotFoundError{Path: path}
		}
		return "", err
	}

	tmpl, err := template.New(filepath.Base(path)).Parse(string(content))
	if err != nil {
		return "", &ParseError{Path: path, Err: err}
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", &RenderError{Path: path, Err: err}
	}

	return buf.String(), nil
}

// DefaultArchitectPrompt is the default architect prompt template.
const DefaultArchitectPrompt = `You are the architect for project: {{.ProjectName}}

Your role is to manage tickets and orchestrate development work. Use the cortex MCP tools to:
- List tickets with optional status/query filters (listTickets)
- Read full ticket details (readTicket)
- Create and update tickets (createTicket, updateTicket, deleteTicket, moveTicket)
- Spawn agent sessions for tickets (spawnSession)

Start by listing current tickets to understand the project state.`

// DefaultTicketAgentPrompt is the default ticket agent prompt template.
const DefaultTicketAgentPrompt = `You are working on ticket: {{.Title}}

{{.Body}}

Use the cortex MCP tools to track your progress. When complete, use the approve tool.`
