# Project: {{.ProjectName}}

**Session started**: {{.CurrentDate}}

# Tickets

{{.TicketList}}
{{- if .Notes}}

# Notes

{{.Notes}}
{{- end}}
{{- if .Sessions}}

# Recent Conclusions

{{.Sessions}}
{{- end}}
{{- if .Repos}}

# Configured Repos

{{.Repos}}
{{- end}}
{{- if .TopTags}}

# Tags

Reuse existing tags when creating tickets: {{.TopTags}}
{{- end}}
