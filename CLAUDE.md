# Cortex - AI Development Workflow

Orchestration layer for AI coding agents. File-based ticket management with MCP tools, tmux session management, and a daemon-centric architecture.

## Build & Test

```bash
make install              # Build, install to ~/.local/bin/, codesign (macOS)
make build                # Build bin/cortex and bin/cortexd only
make lint                 # golangci-lint
make test                 # Unit tests
make test-integration     # Integration tests (requires running daemon)
```

## Architecture

Single `cortexd` daemon serves all projects. **All clients communicate exclusively over HTTP** — no client accesses the ticket store directly. This enables running the daemon on a remote VM with local client TUIs.

```
┌─────────────────┐
│  cortex CLI/TUI │──┐
└─────────────────┘  │
┌─────────────────┐  │  HTTP :4200   ┌──────────────────────┐
│  MCP Architect  │──┼──────────────▶│      cortexd         │
└─────────────────┘  │               │  ├─ HTTP API         │
┌─────────────────┐  │               │  ├─ StoreManager     │
│  MCP Ticket     │──┘               │  ├─ Tmux management  │
└─────────────────┘                  │  └─ SSE event bus    │
                                     └──────────────────────┘
```

Project context: `X-Cortex-Project` header (HTTP) or `CORTEX_PROJECT_PATH` env (MCP).

## Key Paths

| Component | Path |
|-----------|------|
| CLI commands | `cmd/cortex/commands/` |
| Daemon commands | `cmd/cortexd/commands/` |
| HTTP API handlers | `internal/daemon/api/` |
| MCP tools | `internal/daemon/mcp/` |
| Ticket store | `internal/ticket/` |
| SDK client | `internal/cli/sdk/client.go` |
| Spawn orchestration | `internal/core/spawn/` |
| Project config | `internal/project/config/` |
| Daemon config | `internal/daemon/config/` |
| Tmux manager | `internal/tmux/` |
| Worktree manager | `internal/worktree/` |
| TUI components | `internal/cli/tui/` |
| Install/init logic | `internal/install/` |

## Configuration

**Project** (`.cortex/cortex.yaml`): Agent type, agent args, git worktrees, lifecycle hooks. See `internal/project/config/config.go` for schema.

**Global** (`~/.cortex/settings.yaml`): Daemon port, log level, project registry. See `internal/daemon/config/config.go` for schema.

**Project registry**: Global config tracks all projects (`projects` list). Auto-registered on `cortex init`. Used by `GET /projects` endpoint and `cortex projects` CLI.

## CLI Commands

| Command | Description |
|---------|-------------|
| `cortex init` | Initialize `.cortex/` in current directory, register in global config |
| `cortex architect` | Start/attach architect session |
| `cortex kanban` | Kanban TUI for current project |
| `cortex show [id]` | Ticket detail TUI |
| `cortex ticket list` | List tickets |
| `cortex ticket spawn <id>` | Spawn ticket agent session |
| `cortex projects` | List all registered projects with ticket counts |
| `cortex register [path]` | Register project in global config |
| `cortex unregister [path]` | Remove project from global config |

## API Endpoints

Routes defined in `internal/daemon/api/server.go`. SDK client in `internal/cli/sdk/client.go`.

**Global** (no project header): `GET /health`, `GET /projects`

**Project-scoped** (requires `X-Cortex-Project`): Ticket CRUD, spawn, move, comments, reviews, conclude, architect spawn, session kill/approve, SSE events.

## MCP Tools

Defined in `internal/daemon/mcp/`. Two session types:

**Architect** (`tools_architect.go`): `listTickets`, `readTicket`, `createTicket`, `updateTicket`, `deleteTicket`, `moveTicket`, `spawnSession`

**Ticket** (`tools_ticket.go`): `readTicket`, `addTicketComment`, `requestReview`, `concludeSession`

## Agent Workflow

1. Architect reads backlog → calls `spawnSession` for a ticket
2. Daemon creates tmux window with ticket-scoped MCP (30% agent pane, 70% companion pane)
3. Ticket agent works, uses `addTicketComment` to log progress
4. Agent calls `requestReview` when done → ticket moves to review
5. Architect reviews and approves → triggers lifecycle hooks, moves to done

Spawn orchestration handles state detection (normal/active/orphaned/ended) and mode selection (normal/resume/fresh). See `internal/core/spawn/orchestrate.go`.

## Lifecycle Hooks

Defined in `.cortex/cortex.yaml` under `lifecycle`. Hooks run on pickup, review, and approve. Template variables: `{{.Slug}}`, `{{.CommitMessage}}`, etc. See `internal/lifecycle/` for execution logic.

## Testing

- Unit: `make test`
- Integration: `make test-integration`
- API tests: `internal/daemon/api/integration_test.go`
- MCP tests: `internal/daemon/mcp/server_test.go`
- Config tests: `internal/daemon/config/config_test.go`
