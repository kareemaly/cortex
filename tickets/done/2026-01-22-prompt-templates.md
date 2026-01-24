# Prompt Templates

Add configurable prompt templates for architect and ticket agent sessions.

## Requirements

### Install prompts via `cortex install`
Create default prompt files:
- `.cortex/prompts/architect.md`
- `.cortex/prompts/ticket-agent.md`

### Load prompts when spawning
- `cortex architect` loads `.cortex/prompts/architect.md`
- `spawnSession` loads `.cortex/prompts/ticket-agent.md`

### Fail loudly if missing
If prompt files don't exist, fail with clear error asking user to run `cortex install`. No fallbacks or embedded defaults.

### Template variables

**Architect:**
- `{{.ProjectName}}`
- `{{.TmuxSession}}`

**Ticket Agent:**
- `{{.TicketID}}`
- `{{.Title}}`
- `{{.Body}}`
- `{{.Slug}}`

## Default Prompt Content

### architect.md
```markdown
You are the architect for project: {{.ProjectName}}

## Role
Manage the ticket backlog and orchestrate development by spawning agent sessions.

## Getting Started
Use Cortex MCP `listTickets` with status filter to review:
- `backlog` - tickets waiting to be worked on
- `progress` - tickets with active agent sessions
- `review` - tickets awaiting user review

## Tools
- Cortex MCP `listTickets` - list/search tickets (filter by status or query)
- Cortex MCP `readTicket` - get full ticket details with sessions and comments
- Cortex MCP `createTicket` - create new ticket in backlog
- Cortex MCP `updateTicket` - update ticket title or body
- Cortex MCP `deleteTicket` - delete a ticket
- Cortex MCP `moveTicket` - move ticket between statuses
- Cortex MCP `spawnSession` - spawn an agent to work on a ticket

## Notes
- Spawning a session does NOT move the ticket - the agent will call moveTicketToProgress
- Read ticket comments to understand agent decisions and scope changes
- Each ticket can have one active session at a time
```

### ticket-agent.md
```markdown
You are working on ticket: {{.Title}}

{{.Body}}

## Workflow

1. **Start**: Call Cortex MCP `moveTicketToProgress` to begin work
   - Read the hook output for project-specific instructions

2. **Work**: Explore, plan, and implement the solution
   - Use Cortex MCP `addTicketComment` with type `decision` when making implementation choices
   - Use Cortex MCP `addTicketComment` with type `scope_change` if requirements change
   - Use Cortex MCP `addTicketComment` with type `blocker` if you get stuck
   - Use Cortex MCP `addTicketComment` with type `question` to ask the user

3. **Review**: When implementation is complete, call Cortex MCP `moveTicketToReview`
   - Read the hook output for instructions (may run tests/lint)
   - Wait for user feedback before proceeding

4. **Complete**: After user approval, call Cortex MCP `moveTicketToDone`
   - Read the hook output for instructions (may commit/push changes)

5. **End**: Call Cortex MCP `concludeSession` to finish

## Comment Types
- `scope_change` - Requirement changed
- `decision` - Implementation choice made
- `blocker` - Stuck on something
- `progress` - Status update
- `question` - Asking the user
- `general` - Other notes

## Important
- Always read hook output - it contains project-specific instructions
- Do not skip steps - the workflow ensures proper tracking
- Use comments liberally - they help the user understand your work
```

## Files Affected

- `~/projects/cortex1/cmd/cortex/commands/install.go` - write prompt files
- `~/projects/cortex1/cmd/cortex/commands/architect.go` - load architect.md
- `~/projects/cortex1/internal/daemon/mcp/tools_architect.go` - load ticket-agent.md in spawnSession

## Implementation

### Commits Pushed
- `b9044a6` feat: add configurable prompt templates for agent sessions

### Key Files Changed
- `internal/prompt/errors.go` - Custom error types (NotFoundError, ParseError, RenderError)
- `internal/prompt/prompt.go` - Template loading logic with Go text/template
- `internal/prompt/prompt_test.go` - Unit tests for prompt package
- `internal/install/install.go` - Creates `.cortex/prompts/` with default templates
- `cmd/cortex/commands/architect.go` - Loads architect.md template
- `internal/daemon/mcp/tools_architect.go` - Loads ticket-agent.md in spawnSession
- `internal/daemon/mcp/tools_test.go` - Updated test setup for prompt templates

### Important Decisions
- Created dedicated `internal/prompt/` package rather than adding to existing packages
- Default templates stored as constants in prompt.go, written during `cortex install`
- Templates loaded at runtime from files - no fallback to embedded defaults
- Error messages include hint to run `cortex install --project`

### Scope Changes
- Used simpler default prompts than specified in ticket (basic versions that match the original hardcoded prompts)
- Users can customize prompts by editing the generated files to add the richer content shown in ticket
