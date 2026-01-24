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
`

// DefaultTicketPrompt is the template for ticket content.
const DefaultTicketPrompt = `# Ticket: {{.TicketTitle}}

{{.TicketBody}}
`

// DefaultTicketWorktreePrompt includes worktree-specific information.
const DefaultTicketWorktreePrompt = `# Ticket: {{.TicketTitle}}

{{.TicketBody}}

## Worktree Information

- **Path**: {{.WorktreePath}}
- **Branch**: {{.WorktreeBranch}}

All changes should be made in this worktree. The branch will be merged on approval.
`

// DefaultApprovePrompt contains instructions for the approval workflow.
const DefaultApprovePrompt = `## Review Approved

Your changes have been reviewed and approved. Complete the following steps:

1. **Verify all changes are committed**
   - Run ` + "`git status`" + ` to check for uncommitted changes
   - Commit any remaining changes

2. **Push your branch** (if not already pushed)
   - Run ` + "`git push`" + `

3. **Call concludeSession**
   - Call ` + "`mcp__cortex__concludeSession`" + ` with a complete report including:
     - Summary of all changes made
     - Key decisions and their rationale
     - List of files modified
     - Any follow-up tasks or notes

This will mark the ticket as done and end your session.
`

// DefaultApproveWorktreePrompt contains instructions for approving worktree changes.
const DefaultApproveWorktreePrompt = `## Review Approved

Your changes have been reviewed and approved. Complete the following steps:

1. **Verify all changes are committed**
   - Run ` + "`git status`" + ` to check for uncommitted changes
   - Commit any remaining changes

2. **Push your branch**
   - Run ` + "`git push -u origin {{.WorktreeBranch}}`" + `

3. **Merge to main branch** (from the main worktree)
   - The changes in {{.WorktreeBranch}} need to be merged to main
   - This may be done via PR or direct merge depending on project workflow

4. **Call concludeSession**
   - Call ` + "`mcp__cortex__concludeSession`" + ` with a complete report including:
     - Summary of all changes made
     - Key decisions and their rationale
     - List of files modified
     - Any follow-up tasks or notes

This will mark the ticket as done and end your session.
`
