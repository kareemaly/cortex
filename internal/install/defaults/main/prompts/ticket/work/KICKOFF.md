# Ticket: {{.TicketTitle}}

{{.TicketBody}}
{{if .References}}

## References

{{.References}}
{{end}}
{{if .IsWorktree}}

## Worktree Information

- **Path**: {{.WorktreePath}}
- **Branch**: {{.WorktreeBranch}}

All changes should be made in this worktree. The branch will be merged on approval.
{{end}}
{{if .Comments}}

## Comments

{{.Comments}}
{{end}}
