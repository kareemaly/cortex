# Ticket: {{.TicketTitle}}

{{.TicketBody}}
{{if .References}}

## Referenced Tickets

The following tickets are referenced. Use `readTicket` to pull full details on any that are relevant to your work.

{{.References}}
{{end}}
