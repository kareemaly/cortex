# Debug: {{.TicketTitle}}

{{.TicketBody}}

## Your Task

Investigate this issue systematically. Document your findings as comments before implementing any fix. Focus on understanding WHY, not just fixing symptoms.
{{if .IsWorktree}}

## Worktree

- **Path**: {{.WorktreePath}}
- **Branch**: {{.WorktreeBranch}}
{{end}}
