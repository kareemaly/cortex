# Ticket: {{.TicketTitle}}

{{.TicketBody}}
{{if .References}}

## References

{{.References}}
{{end}}
{{if .Comments}}

## Comments

{{.Comments}}
{{end}}

## Workflow

You are working in this ticket to implement the requirements above. Follow this workflow:

1. **Create a branch**: Start with a descriptive branch name (e.g., `feat/add-webhook-support`, `fix/tmux-panes`)

2. **Make conventional commits**: Use conventional commit format:
   - `feat:` for new features
   - `fix:` for bug fixes
   - `docs:` for documentation changes
   - `refactor:` for code refactoring
   - `test:` for test changes

3. **Push when done**: Push your changes to the remote repository

4. **Call concludeSession**: When you've completed the work, call `concludeSession` with a summary of what you did

## Repository Context

{{if .Repo}}
You are working in: {{.Repo}}
{{else}}
You are working in the architect project root
{{end}}

The repo's own documentation (AGENTS.md or CLAUDE.md) may provide repo-specific context and conventions. Read it if available.

## Completion

When you're done:
1. Push your changes to the remote
2. Call `concludeSession` with a summary of:
   - What was implemented
   - Any technical decisions made
   - Files or components affected
   - Testing performed
