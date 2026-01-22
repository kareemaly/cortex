# Cortex - AI Development Workflow

## Build & Install

```bash
make build                    # Build bin/cortex and bin/cortexd
make lint                     # Run golangci-lint
make test                     # Unit tests
make test-integration         # Integration tests
```

**Install globally:**
```bash
cp bin/cortex bin/cortexd ~/.local/bin/
codesign --force --sign - ~/.local/bin/cortex ~/.local/bin/cortexd  # macOS only
```

## Architecture

Single `cortexd` daemon serves ALL projects simultaneously via two interfaces:

1. **HTTP API** (port 4200) - Used by `cortex` CLI and kanban TUI
2. **MCP over stdio** - Used by AI agents (claude/opencode)

Project context is passed via `X-Cortex-Project` header (HTTP) or environment variables (MCP).

```
┌─────────────────┐     ┌─────────────────┐
│  cortex CLI     │────▶│                 │
│  (kanban, list) │ HTTP│    cortexd      │
└─────────────────┘     │                 │
                        │  StoreManager   │
┌─────────────────┐     │  (per-project)  │
│  AI Architect   │────▶│                 │
│  (claude)       │ MCP │                 │
└─────────────────┘     └─────────────────┘
       │
       │ spawns
       ▼
┌─────────────────┐
│  Ticket Agent   │────▶ MCP (restricted to assigned ticket)
│  (claude)       │
└─────────────────┘
```

## Key Paths

| Component | Path |
|-----------|------|
| CLI commands | `~/projects/cortex1/cmd/cortex/commands/` |
| Daemon commands | `~/projects/cortex1/cmd/cortexd/commands/` |
| HTTP API handlers | `~/projects/cortex1/internal/daemon/api/` |
| MCP tools | `~/projects/cortex1/internal/daemon/mcp/` |
| Ticket store | `~/projects/cortex1/internal/ticket/` |
| SDK client | `~/projects/cortex1/internal/cli/sdk/client.go` |
| Project config | `~/projects/cortex1/internal/project/config/` |
| Tmux manager | `~/projects/cortex1/internal/tmux/tmux.go` |

## Project Structure

Each project has `.cortex/` containing:
- `cortex.yaml` - Agent type, git repos, lifecycle hooks
- `tickets/{backlog,progress,done}/` - Ticket markdown files

Global config at `~/.cortex/settings.yaml` (daemon port, log level).

## MCP Tools

**Architect session** (full access, 7 tools):
- `listTickets` - List tickets with optional status/query filters
- `readTicket` - Read full ticket details
- `createTicket`, `updateTicket`, `deleteTicket`, `moveTicket`
- `spawnSession` - Spawn agent session for a ticket

**Ticket session** (restricted to assigned ticket):
- `readTicket` - Read assigned ticket
- `pickupTicket` - Move to in_progress, runs `on_pickup` hooks
- `submitReport` - Update report (files, decisions, summary)
- `approve` - End session, move to done, runs `on_approve` hooks

## Agent Workflow

1. Architect reads backlog tickets via `listTickets`
2. Architect calls `spawnSession` for a ticket
3. Daemon spawns tmux window with ticket-scoped MCP
4. Ticket agent calls `pickupTicket` to start work
5. Agent uses `submitReport` to log progress
6. Agent calls `approve` when done (triggers hooks, moves to done)

## Lifecycle Hooks

Defined in `.cortex/cortex.yaml`:
```yaml
lifecycle:
  on_pickup: ["git checkout -b ticket/{{.Slug}}"]
  on_approve: ["git add -A", "git commit -m '{{.CommitMessage}}'"]
```

## Testing

- Unit tests: `make test`
- Integration tests: `make test-integration` (requires `integration` build tag)
- API tests: `~/projects/cortex1/internal/daemon/api/integration_test.go`
- MCP tests: `~/projects/cortex1/internal/daemon/mcp/server_test.go`
