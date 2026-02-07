---
id: dc6b6b75-b999-4a95-900a-73e35482ad15
author: claude
type: comment
created: 2026-02-05T10:01:00.241789Z
---
## Design Decision: Copilot Agent Prompt Strategy

### Confirmed Approach
For Copilot agents (both architect and ticket), we will:
- **NOT use SYSTEM.md** at all
- **Only use KICKOFF.md** + MCP tool descriptions

### Prompt File Structure

**Claude agents** (current behavior):
```
.cortex/prompts/
├── architect/
│   ├── SYSTEM.md      ✓ Used
│   └── KICKOFF.md     ✓ Used
└── ticket/
    └── {type}/
        ├── SYSTEM.md  ✓ Used
        └── KICKOFF.md ✓ Used
```

**Copilot agents** (new behavior):
```
.cortex/prompts/
├── architect/
│   ├── SYSTEM.md      ✗ Ignored
│   └── KICKOFF.md     ✓ Used (needs MCP workflow details)
└── ticket/
    └── {type}/
        ├── SYSTEM.md  ✗ Ignored
        └── KICKOFF.md ✓ Used (needs MCP workflow details)
```

### KICKOFF.md for Copilot Must Include

Since there's no SYSTEM.md, the kickoff needs to cover:
1. **MCP tool inventory** - What tools are available
2. **Workflow guidance** - When to use each tool
3. **Task context** - Ticket details, project info (already there)

Example structure for ticket agent kickoff:
```markdown
# Your Task
{{.TicketTitle}}

{{.TicketBody}}

# Cortex MCP Tools
You have access to these tools via the `cortex` MCP server:
- `readTicket` - Read your assigned ticket details
- `addComment` - Log progress as you work
- `addBlocker` - Report if you're blocked
- `requestReview` - Call when ready for human review
- `concludeSession` - Call after review is approved

# Workflow
1. Read the ticket and understand requirements
2. Implement the solution, adding comments as you progress
3. Call `requestReview` when done - wait for approval
4. Call `concludeSession` after approval
```

### Implementation Changes

1. `internal/core/spawn/launcher.go` - Skip system prompt for Copilot
2. `.cortex/defaults/prompts/` - Create Copilot-specific KICKOFF.md templates (or make existing ones agent-agnostic with richer content)
3. `internal/core/spawn/spawn.go` - Don't load SYSTEM.md when agent is Copilot