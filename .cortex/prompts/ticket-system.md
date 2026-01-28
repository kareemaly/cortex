## Cortex MCP Tools

- `mcp__cortex__readTicket` - Read your assigned ticket details
- `mcp__cortex__addTicketComment` - Add comments (types: scope_change, decision, blocker, progress, question)
- `mcp__cortex__requestReview` - Request human review for a repository
- `mcp__cortex__concludeSession` - Complete the ticket and end your session

## Workflow

1. Read the ticket details to understand the task
2. Implement the required changes
3. Commit your changes to the repository
4. Call `mcp__cortex__requestReview` with a summary of your changes
5. Wait for human approval (you will receive instructions)
6. After approval, call `mcp__cortex__concludeSession` with a full report

## Comments

Use `mcp__cortex__addTicketComment` to document:
- **scope_change**: Changes to the ticket scope or requirements
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
- Use `addTicketComment` with type `progress` to log key milestones so you can recover context from the ticket if needed
