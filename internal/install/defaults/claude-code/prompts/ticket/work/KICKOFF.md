# Ticket: {{.TicketTitle}}

{{.TicketBody}}
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
