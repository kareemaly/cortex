# Project: {{.ProjectName}}

**Session started**: {{.CurrentDate}}

# Tickets

{{.TicketList}}
{{- if .DocsList}}

# Recent Docs

If there are session docs below, start by reading the most recent one with `readDoc` to pick up context from the last session.

{{.DocsList}}
{{- end}}
{{- if .TopTags}}

# Tags

Reuse existing tags when creating tickets: {{.TopTags}}
{{- end}}
