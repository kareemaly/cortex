package install

// DefaultTicketSystemPrompt contains MCP tool instructions and workflow guidance.
const DefaultTicketSystemPrompt = `## Cortex Workflow

Use Cortex MCP tools: ` + "`readTicket`" + `, ` + "`addComment`" + `, ` + "`addBlocker`" + `, ` + "`requestReview`" + `, ` + "`concludeSession`" + `.

1. Understand the ticket provided below
2. Ask clarifying questions if requirements are unclear
3. Implement the changes
4. Call ` + "`requestReview`" + ` with a summary and wait for user instructions
`

// DefaultTicketKickoffPrompt is the unified template for ticket content.
// Uses {{if .IsWorktree}} conditionals instead of separate worktree files.
const DefaultTicketKickoffPrompt = `# Ticket: {{.TicketTitle}}

{{.TicketBody}}
{{if .IsWorktree}}

## Worktree Information

- **Path**: {{.WorktreePath}}
- **Branch**: {{.WorktreeBranch}}

All changes should be made in this worktree. The branch will be merged on approval.
{{end}}
`

// DefaultTicketApprovePrompt contains instructions for the approval workflow.
const DefaultTicketApprovePrompt = `## Approved

1. Commit your changes
2. Push to origin
3. Call ` + "`concludeSession`" + ` with a summary of what was done
`
