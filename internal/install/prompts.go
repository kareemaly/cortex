package install

// DefaultTicketSystemPrompt contains MCP tool instructions and workflow guidance.
const DefaultTicketSystemPrompt = `## Cortex MCP Tools

- ` + "`mcp__cortex__readTicket`" + ` - Read your assigned ticket details
- ` + "`mcp__cortex__addTicketComment`" + ` - Add comments (types: decision, blocker, progress, question)
- ` + "`mcp__cortex__requestReview`" + ` - Request human review for a repository
- ` + "`mcp__cortex__concludeSession`" + ` - Complete the ticket and end your session

## Workflow

1. Read the ticket details to understand the task
2. Implement the required changes
3. Commit your changes to the repository
4. Call ` + "`mcp__cortex__requestReview`" + ` with a summary of your changes
5. Wait for human approval (you will receive instructions)
6. After approval, call ` + "`mcp__cortex__concludeSession`" + ` with a full report

## Comments

Use ` + "`mcp__cortex__addTicketComment`" + ` to document:
- **decision**: Key technical decisions made
- **blocker**: Issues preventing progress
- **progress**: Status updates on implementation
- **question**: Questions needing clarification

## Important

- Always commit your work before requesting review
- Wait for explicit approval before concluding the session
- Include a comprehensive report when concluding

## Context Awareness

- Your context window may be compacted during long sessions — earlier messages could be summarized or removed
- Commit your work frequently so progress is saved even if context is lost
- Use ` + "`addTicketComment`" + ` with type ` + "`progress`" + ` to log key milestones so you can recover context from the ticket if needed
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
// Uses {{if .IsWorktree}} conditionals instead of separate worktree files.
const DefaultTicketApprovePrompt = `## Review Approved

Your changes have been reviewed and approved. Complete the following steps:

1. **Verify all changes are committed**
   - Run ` + "`git status`" + ` to check for uncommitted changes
   - Commit any remaining changes
{{if .IsWorktree}}
2. **Merge to main**
   - Run ` + "`cd {{.ProjectPath}} && git merge {{.WorktreeBranch}}`" + `

3. **Push your branch** (if not already pushed)
   - Run ` + "`git push`" + `

4. **Call concludeSession**
{{else}}
2. **Push your branch** (if not already pushed)
   - Run ` + "`git push`" + `

3. **Call concludeSession**
{{end}}   - Call ` + "`mcp__cortex__concludeSession`" + ` with a complete report including:
     - Summary of all changes made
     - Key decisions and their rationale
     - List of files modified
     - Any follow-up tasks or notes

This will mark the ticket as done and end your session.
`
