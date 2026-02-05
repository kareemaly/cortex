# Cortex - AI Development Workflow

Orchestration layer for AI coding agents. File-based ticket management with MCP tools, tmux session management, and a daemon-centric architecture.

## Quick Start

```bash
cortexd &                 # Start daemon (background)
cortex init               # Initialize project
cortex architect          # Start architect session
```

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

## Critical Implementation Notes

- **HTTP-only communication**: All clients (CLI, TUI, MCP) communicate via HTTP to daemon. No direct filesystem access to ticket store.
- **Project context**: Always use `X-Cortex-Project` header (HTTP) or `CORTEX_PROJECT_PATH` env (MCP).
- **StoreManager**: Single source of truth for ticket state. Located in `internal/daemon/api/store_manager.go`.
- **Spawn state detection**: Four states (normal/active/orphaned/ended) with mode matrix (normal/resume/fresh). See `internal/core/spawn/orchestrate.go`.

## Anti-Patterns

| Don't | Do Instead | Why |
|-------|------------|-----|
| Access ticket JSON files directly | Use SDK client (`internal/cli/sdk/client.go`) | Daemon holds in-memory state with locks |
| Spawn tmux sessions directly | Use `SpawnSession()` via SDK/API | Bypasses session tracking and MCP binding |
| Import `internal/ticket` in CLI code | Use HTTP API endpoints | Breaks daemon-as-authority architecture |
| Import `internal/core/spawn` in CLI | Call `/tickets/{status}/{id}/spawn` | CLI should not import daemon internals |

## Debugging

| Symptom | Check |
|---------|-------|
| Daemon not responding | `cortex daemon status`, verify port 4200, check `~/.cortex/logs/` |
| Ticket not found | Verify `X-Cortex-Project` header matches project path |
| Session won't spawn | Check `cortex ticket list` for state, use `mode=resume` for orphaned |
| MCP tools not working | Verify `CORTEX_PROJECT_PATH` env, check daemon logs |

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
| `cortex daemon status` | Check daemon status |
| `cortex upgrade` | Self-update to latest version |
| `cortex eject <path>` | Customize a default prompt |

## API Endpoints

Routes defined in `internal/daemon/api/server.go`. SDK client in `internal/cli/sdk/client.go`.

**Global** (no project header): `GET /health`, `GET /projects`

**Project-scoped** (requires `X-Cortex-Project`): Ticket CRUD, spawn, move, comments, reviews, conclude, architect spawn, session kill/approve, SSE events.

## MCP Tools

Defined in `internal/daemon/mcp/`. Two session types with different tool access:

**Architect** (`tools_architect.go`):

| Tool | Description |
|------|-------------|
| `listTickets` | List tickets by status (backlog/progress/review/done), optional search query |
| `readTicket` | Read full ticket details by ID |
| `createTicket` | Create ticket with title, body, type |
| `updateTicket` | Update ticket title and/or body |
| `deleteTicket` | Delete ticket by ID |
| `moveTicket` | Move ticket to different status |
| `addTicketComment` | Add comment to ticket (types: review_requested, done, blocker, comment) |
| `spawnSession` | Spawn agent session for ticket (modes: normal, resume, fresh) |
| `getCortexConfigDocs` | Get CONFIG_DOCS.md for customization guidance |

**Ticket Agent** (`tools_ticket.go`):

| Tool | Description |
|------|-------------|
| `readTicket` | Read assigned ticket details (no ID required) |
| `addComment` | Add comment to assigned ticket |
| `addBlocker` | Report blocker on assigned ticket |
| `requestReview` | Request human review, moves ticket to review status |
| `concludeSession` | Complete work, move to done, trigger cleanup |

## Agent Workflow

1. Architect reads backlog → calls `spawnSession` for a ticket
2. Daemon creates tmux window with ticket-scoped MCP (30% agent pane, 70% companion pane)
3. Ticket agent works, uses `addComment` to log progress
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
