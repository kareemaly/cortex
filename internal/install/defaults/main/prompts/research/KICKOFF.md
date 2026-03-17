# Ticket: {{.TicketTitle}}

{{.TicketBody}}
{{if .References}}

## Referenced Tickets

The following tickets are referenced. Use `readTicket` to pull full details on any that are relevant to your work.

{{.References}}
{{end}}

## Guidelines

Explore thoroughly, then discuss your findings with the user before taking any further action.

## Conclusion

When you call `concludeSession`, include the actual outcome of the session:
- What you found, changed, or accomplished
- The files you modified, if any
- The commit SHA, if you created a commit
- Any important follow-up work, blockers, or caveats
