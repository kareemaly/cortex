---
id: 72ae24ce-331c-4ea6-bc8f-69c1cba804ac
title: MCP Server
type: ""
created: 2026-01-20T13:06:22Z
updated: 2026-01-20T13:06:22Z
---
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

## Implementation

### Commits

- `037fb93` feat: add MCP server for AI agent integration with ticket management tools

### Key Files Changed

**New files in `internal/daemon/mcp/`:**
- `types.go` - Input/output structs with jsonschema tags for schema generation
- `errors.go` - ToolError type with error codes (NOT_FOUND, VALIDATION_ERROR, UNAUTHORIZED, INTERNAL_ERROR)
- `server.go` - MCP server setup, session management, tool registration
- `tools_architect.go` - 9 architect tools (listTickets, searchTickets, readTicket, createTicket, updateTicket, deleteTicket, moveTicket, spawnSession, getSessionStatus)
- `tools_ticket.go` - 4 ticket tools (readTicket, pickupTicket, submitReport, approve)
- `server_test.go` - Server initialization tests
- `tools_test.go` - Comprehensive tool handler tests (26 tests)

**New files in `cmd/cortexd/commands/`:**
- `root.go` - Root Cobra command for cortexd
- `serve.go` - HTTP server subcommand (default behavior)
- `mcp.go` - MCP server subcommand with `--ticket-id` flag

**Modified:**
- `cmd/cortexd/main.go` - Simplified to call `commands.Execute()`
- `go.mod` / `go.sum` - Added `github.com/modelcontextprotocol/go-sdk` v1.2.0

### Decisions

1. **Tool registration at startup**: Tools are registered based on session type at server creation, not filtered at runtime
2. **Struct tags for schemas**: Using `json:"field"` and `jsonschema:"description"` tags for automatic JSON schema generation per MCP SDK conventions
3. **spawnSession stub**: Returns "not implemented" message since tmux integration is a separate ticket
4. **Cobra subcommands**: Refactored cortexd to use Cobra pattern matching `cmd/cortex/commands/` structure
5. **Default behavior**: Running `cortexd` without subcommand defaults to `serve` (HTTP server) for backward compatibility

### Scope

No scope changes - implemented all requirements as specified.