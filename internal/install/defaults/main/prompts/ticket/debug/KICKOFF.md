# Debug: {{.TicketTitle}}

{{.TicketBody}}
{{if .References}}

## References

{{.References}}
{{end}}

## Your Task

Investigate this issue systematically. Document your findings as comments before implementing any fix. Focus on understanding WHY, not just fixing symptoms.
{{if .IsWorktree}}

## Worktree

- **Path**: {{.WorktreePath}}
- **Branch**: {{.WorktreeBranch}}
{{end}}
{{if .Comments}}

## Comments

{{.Comments}}
{{end}}
