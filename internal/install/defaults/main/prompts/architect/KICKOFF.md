# Project: {{.ProjectName}}

**Session started**: {{.CurrentDate}}

# Tickets

{{.TicketList}}
{{- if .Sessions}}

# Recent Conclusions

{{.Sessions}}
{{- end}}
{{- if .Repos}}

# Configured Repos

{{.Repos}}
{{- end}}
