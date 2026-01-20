# MCP Server

Implement the MCP server that exposes tools for AI agents to interact with tickets.

## Context

Agents interact with cortex via MCP tools, not REST API. The daemon runs an MCP server (stdio transport) that Claude connects to via `--mcp-config`.

See `DESIGN.md` for:
- MCP tools table (lines 171-202)
- Tool parameters and descriptions
- Hook response format (lines 206-226)
- MCP config file format (lines 383-396)
- Agent spawning examples (lines 356-378)

## Requirements

Create `internal/daemon/mcp/` package that:

1. **MCP Server** using `github.com/modelcontextprotocol/go-sdk`
   - Stdio transport for Claude integration
   - Tool registration framework

2. **Architect Session Tools**
   - `listTickets` - List tickets, optionally filter by status
   - `searchTickets` - Search by title, keyword, date
   - `readTicket` - Read full ticket content
   - `createTicket` - Create ticket in backlog
   - `updateTicket` - Update ticket title/body
   - `deleteTicket` - Delete ticket
   - `moveTicket` - Move ticket between statuses
   - `spawnSession` - Start tmux + agent for ticket (stub tmux for now)
   - `getSessionStatus` - Get agent status for ticket

3. **Ticket Session Tools**
   - `readTicket` - Read own ticket (uses env var for ticket ID)
   - `pickupTicket` - Signal starting work
   - `submitReport` - Update session report
   - `approve` - Approve and conclude ticket

4. **Integration**
   - Wire MCP server into daemon (new subcommand or mode)
   - Use `internal/ticket` store for all operations
   - Return structured responses per DESIGN.md

## Verification

```bash
make build   # Builds successfully
make test    # Tests pass
make lint    # No lint errors
```

## Notes

- Architect and ticket sessions get different tool sets
- Ticket session tools use CORTEX_TICKET_ID env var
- Stub tmux/lifecycle hooks for now - those are separate tickets
- Focus on tool registration and ticket store integration
