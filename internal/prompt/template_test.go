package prompt

import (
	"strings"
	"testing"
)

func TestRenderTemplate_ArchitectKickoff_WithTags(t *testing.T) {
	tmpl := `# Project: {{.ProjectName}}
{{.TicketList}}
{{- if .TopTags}}

# Tags

Reuse existing tags: {{.TopTags}}
{{- end}}`

	vars := ArchitectKickoffVars{
		ProjectName: "TestProject",
		TicketList:  "## Backlog\n- [t1] Task 1\n",
		CurrentDate: "2025-06-01 10:00 UTC",
		TopTags:     "api, bug, feature",
	}

	result, err := RenderTemplate(tmpl, vars)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "# Tags") {
		t.Error("expected tags section to be present")
	}
	if !strings.Contains(result, "api, bug, feature") {
		t.Error("expected tags to be present")
	}
}

func TestRenderTemplate_ArchitectKickoff_EmptyTags(t *testing.T) {
	tmpl := `# Project: {{.ProjectName}}
{{.TicketList}}
{{- if .TopTags}}

# Tags

Reuse existing tags: {{.TopTags}}
{{- end}}`

	vars := ArchitectKickoffVars{
		ProjectName: "TestProject",
		TicketList:  "## Backlog\n(none)\n",
		CurrentDate: "2025-06-01 10:00 UTC",
		TopTags:     "",
	}

	result, err := RenderTemplate(tmpl, vars)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(result, "# Tags") {
		t.Error("expected tags section to be omitted when empty")
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
