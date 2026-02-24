package prompt

import (
	"strings"
	"testing"
)

func TestRenderTemplate_ArchitectKickoff_Basic(t *testing.T) {
	tmpl := `# Project: {{.ArchitectName}}
{{.TicketList}}
{{- if .Sessions}}

# Recent Conclusions

{{.Sessions}}
{{- end}}`

	vars := ArchitectKickoffVars{
		ArchitectName: "TestProject",
		TicketList:    "## Backlog\n- [t1] Task 1\n",
		CurrentDate:   "2025-06-01 10:00 UTC",
	}

	result, err := RenderTemplate(tmpl, vars)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "# Project: TestProject") {
		t.Error("expected project name to be present")
	}
	if !strings.Contains(result, "Task 1") {
		t.Error("expected ticket list to be present")
	}
	if strings.Contains(result, "# Recent Conclusions") {
		t.Error("expected conclusions section to be omitted when empty")
	}
}

func TestRenderTemplate_TicketKickoff_WithReferences(t *testing.T) {
	tmpl := `# Ticket: {{.TicketTitle}}

{{.TicketBody}}
{{if .References}}

## References

{{.References}}
{{end}}`

	vars := TicketVars{
		TicketTitle: "Test Ticket",
		TicketBody:  "Some ticket body",
		References:  "- ticket:abc123\n- doc:xyz789",
	}

	result, err := RenderTemplate(tmpl, vars)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "## References") {
		t.Error("expected references section to be present")
	}
	if !strings.Contains(result, "- ticket:abc123") {
		t.Error("expected first reference to be present")
	}
	if !strings.Contains(result, "- doc:xyz789") {
		t.Error("expected second reference to be present")
	}
}

func TestRenderTemplate_TicketKickoff_EmptyReferences(t *testing.T) {
	tmpl := `# Ticket: {{.TicketTitle}}

{{.TicketBody}}
{{if .References}}

## References

{{.References}}
{{end}}`

	vars := TicketVars{
		TicketTitle: "Test Ticket",
		TicketBody:  "Some ticket body",
		References:  "",
	}

	result, err := RenderTemplate(tmpl, vars)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(result, "## References") {
		t.Error("expected references section to be omitted when empty")
	}
}
