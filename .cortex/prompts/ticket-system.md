## Cortex MCP Tools

- `mcp__cortex__readTicket` - Read your assigned ticket details
- `mcp__cortex__addTicketComment` - Add comments (types: decision, blocker, progress, question)
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
- **decision**: Key technical decisions made
- **blocker**: Issues preventing progress
- **progress**: Status updates on implementation
- **question**: Questions needing clarification

## Important

- Always commit your work before requesting review
- Wait for explicit approval before concluding the session
- Include a comprehensive report when concluding
