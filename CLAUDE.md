# Cortex - AI Development Workflow

Orchestration layer for AI coding agents. File-based ticket management with MCP tools, tmux session management, and a daemon-centric architecture.

## Quick Start

```bash
cortexd &                           # Start daemon (background)
cortex architect create --name myproject   # Create architect workspace
cortex architect start              # Start architect session
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
│  MCP Architect  │──┤──────────────▶│      cortexd         │
└─────────────────┘  │               │  ├─ HTTP API         │
┌─────────────────┐  │               │  ├─ StoreManager     │
│  MCP Ticket     │──┘               │  ├─ Tmux management  │
└─────────────────┘                  │  └─ SSE event bus    │
                                     └──────────────────────┘
```

Two-tier agent hierarchy: **Architect** (architect-scoped) → **Ticket Agent** (ticket-scoped).

Architect context: `X-Cortex-Architect` header (HTTP) or `CORTEX_ARCHITECT_PATH` env (MCP).

## Storage Format

Tickets and conclusions use **YAML frontmatter + markdown body** stored as `index.md` within a directory-per-entity:

- **Tickets**: `tickets/{status}/{slug}-{shortid}/index.md` (statuses: backlog, progress, done)
- **Conclusions**: `sessions/{slug}-{shortid}/index.md` (persistent session records)

Default ticket path is `{projectRoot}/tickets/` (configurable via `tickets.path` in `cortex.yaml`). Sessions are ephemeral and stored in `.cortex/sessions.json`.

## Critical Implementation Notes

- **HTTP-only communication**: All clients (CLI, TUI, MCP) communicate via HTTP to daemon. No direct filesystem access to ticket store.
- **Architect context**: Always use `X-Cortex-Architect` header (HTTP) or `CORTEX_ARCHITECT_PATH` env (MCP).
- **StoreManager**: Single source of truth for ticket state. Located in `internal/daemon/api/store_manager.go`.
- **ConclusionStoreManager**: Manages conclusion stores per architect. Located in `internal/daemon/api/conclusion_store_manager.go`.
- **SessionManager**: Manages ephemeral session stores per architect. Located in `internal/daemon/api/session_manager.go`.
- **Spawn state detection**: Three states (normal/active/orphaned) with mode matrix (normal/resume/fresh). See `internal/core/spawn/orchestrate.go`.

## Anti-Patterns

| Don't | Do Instead | Why |
|-------|------------|-----|
| Access ticket files directly | Use SDK client (`internal/cli/sdk/client.go`) | Daemon holds in-memory state with locks |
| Spawn tmux sessions directly | Use `SpawnSession()` via SDK/API | Bypasses session tracking and MCP binding |
| Import `internal/ticket` in CLI code | Use HTTP API endpoints | Breaks daemon-as-authority architecture |
| Import `internal/core/spawn` in CLI | Call `/tickets/{status}/{id}/spawn` | CLI should not import daemon internals |
| Access `.cortex/sessions.json` directly | Use SessionManager API | Bypasses locking and event notifications |
| Import `internal/architect` in CLI code | Use HTTP API endpoints | Breaks daemon-as-authority architecture |

## Debugging

| Symptom | Check |
|---------|-------|
| Daemon not responding | `cortex daemon status`, verify port 4200, check `~/.cortex/logs/` |
| Ticket not found | Verify `X-Cortex-Architect` header matches architect path |
| Session won't spawn | Check `cortex architect show` for state, use `mode=resume` for orphaned |
| MCP tools not working | Verify `CORTEX_ARCHITECT_PATH` env, check daemon logs |

## Key Paths

| Component | Path |
|-----------|------|
| CLI commands | `cmd/cortex/commands/` |
| Daemon commands | `cmd/cortexd/commands/` |
| HTTP API handlers | `internal/daemon/api/` |
| MCP tools | `internal/daemon/mcp/` |
| Ticket store | `internal/ticket/` |
| Conclusion store | `internal/conclusion/` |
| SDK client | `internal/cli/sdk/client.go` |
| Spawn orchestration | `internal/core/spawn/` |
| Architect config | `internal/architect/config/` |
| Daemon config | `internal/daemon/config/` |
| Tmux manager | `internal/tmux/` |
| TUI components | `internal/cli/tui/` (`views/` wrapper, `kanban/`, `sessions/`, `config/`, `ticket/`) |
| Install/init logic | `internal/install/` |
| Agent defaults | `internal/install/defaults/main/` (shared prompts for all agents) |
| Shared storage | `internal/storage/` |
| Session store | `internal/session/` |
| Response types | `internal/types/` |

## Configuration

**Architect** (`cortex.yaml` or `.cortex/cortex.yaml`): Agent type (`claude`, `opencode`) and args per role. Optional `repos` list and `tickets.path`. See `internal/architect/config/config.go` for schema.

```yaml
name: my-project
repos:
  - /path/to/repo
architect:
  agent: claude
work:
  agent: claude
research:
  agent: claude
tickets:
  path: custom/tickets  # optional, defaults to {architectRoot}/tickets
```

**Global** (`~/.cortex/settings.yaml`): Daemon port, bind address (default `127.0.0.1`), log level, architect registry. See `internal/daemon/config/config.go` for schema.

**Architect registry**: Global config tracks all architects (`architects` list). Auto-registered on `cortex architect create`. Used by `GET /architects` endpoint.

## CLI Commands

| Command | Description |
|---------|-------------|
| `cortex architect create [--name <n>] [--agent claude\|opencode]` | Create architect workspace in `~/architects/<name>/` |
| `cortex architect list` | List all registered architects (TUI or table) |
| `cortex architect start [name] [--mode fresh\|resume]` | Start/attach architect session |
| `cortex architect show [name]` | Open architect project TUI |
| `cortex architect delete <name>` | Unlink and optionally delete architect workspace |
| `cortex daemon status` | Check daemon status |
| `cortex upgrade` | Refresh `~/.cortex/defaults/` with latest embedded defaults |
| `cortex eject <path>` | Customize a default prompt |

## API Endpoints

Routes defined in `internal/daemon/api/server.go`. SDK client in `internal/cli/sdk/client.go`.

**Global** (no architect header): `GET /health`, `GET /architects`, `POST /architects`, global config (`/config/global`), daemon logs/status (`/daemon/logs`, `/daemon/status`).

**Architect-scoped** (requires `X-Cortex-Architect`): Ticket CRUD, spawn, move, conclude, architect spawn/conclude, session kill/approve, SSE events, conclusions (`/conclusions`), architect config (`/config/project`, `/config/project/edit`), prompts (`/prompts`, `/prompts/resolve`, `/prompts/eject`, `/prompts/edit`, `/prompts/reset`).

## MCP Tools

Defined in `internal/daemon/mcp/`. Two session types with different tool access:

**Architect** (`tools_architect.go`):

| Tool | Description |
|------|-------------|
| `listTickets` | List tickets by status (backlog/progress/done), optional search query |
| `readTicket` | Read full ticket details by ID |
| `createWorkTicket` | Create a work ticket with title, body, required repo, optional due_date and references |
| `createResearchTicket` | Create a research ticket with title, body, optional due_date and references (no repo) |
| `updateTicket` | Update ticket title, body, and/or references |
| `deleteTicket` | Delete ticket by ID |
| `moveTicket` | Move ticket to different status |
| `updateDueDate` | Set or update ticket due date |
| `clearDueDate` | Remove due date from ticket |
| `spawnSession` | Spawn agent session for ticket (modes: normal, resume, fresh) |
| `listSessions` | List persistent conclusions (session records) |
| `readSession` | Read a conclusion by ID |
| `concludeSession` | Conclude the architect session and clean up |

**Ticket Agent** (`tools_ticket.go`):

| Tool | Description |
|------|-------------|
| `concludeSession` | Complete work, create conclusion record, move ticket to done |

## Agent Workflow

1. Architect reads backlog → calls `spawnSession` for a ticket
2. Daemon creates tmux window with ticket-scoped MCP (30% agent pane, 70% companion pane)
3. Ticket agent works autonomously
4. Agent calls `concludeSession` when done → conclusion record created, ticket moved to done

Spawn orchestration handles state detection (normal/active/orphaned) and mode selection (normal/resume/fresh). See `internal/core/spawn/orchestrate.go`. Architect and ticket agent sessions are tracked in `.cortex/sessions.json`.

## Testing

- Unit: `make test`
- Integration: `make test-integration`
- API tests: `internal/daemon/api/integration_test.go`
- MCP tests: `internal/daemon/mcp/server_test.go`
- Config tests: `internal/daemon/config/config_test.go`
- Architect config tests: `internal/architect/config/config_test.go`
