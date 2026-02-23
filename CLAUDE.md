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
│  MCP Architect  │──┤──────────────▶│      cortexd         │
└─────────────────┘  │               │  ├─ HTTP API         │
┌─────────────────┐  │               │  ├─ StoreManager     │
│  MCP Ticket     │──┘               │  ├─ Tmux management  │
└─────────────────┘                  │  └─ SSE event bus    │
                                     └──────────────────────┘
```

Two-tier agent hierarchy: **Architect** (project-scoped) → **Ticket Agent** (ticket-scoped).

Project context: `X-Cortex-Project` header (HTTP) or `CORTEX_PROJECT_PATH` env (MCP).

## Storage Format

Tickets and conclusions use **YAML frontmatter + markdown body** stored as `index.md` within a directory-per-entity:

- **Tickets**: `tickets/{status}/{slug}-{shortid}/index.md` (statuses: backlog, progress, done)
- **Conclusions**: `sessions/{slug}-{shortid}/index.md` (persistent session records)

Default ticket path is `{projectRoot}/tickets/` (configurable via `tickets.path` in `cortex.yaml`). Sessions are ephemeral and stored in `.cortex/sessions.json`.

Notes use a **single YAML file** (`{projectRoot}/notes.yaml`) — lightweight reminders surfaced in the architect kickoff prompt.

## Critical Implementation Notes

- **HTTP-only communication**: All clients (CLI, TUI, MCP) communicate via HTTP to daemon. No direct filesystem access to ticket store.
- **Project context**: Always use `X-Cortex-Project` header (HTTP) or `CORTEX_PROJECT_PATH` env (MCP).
- **StoreManager**: Single source of truth for ticket state. Located in `internal/daemon/api/store_manager.go`.
- **ConclusionStoreManager**: Manages conclusion stores per project. Located in `internal/daemon/api/conclusion_store_manager.go`.
- **NotesStoreManager**: Manages note stores per project. Located in `internal/daemon/api/notes_store_manager.go`.
- **SessionManager**: Manages ephemeral session stores per project. Located in `internal/daemon/api/session_manager.go`.
- **Spawn state detection**: Three states (normal/active/orphaned) with mode matrix (normal/resume/fresh). See `internal/core/spawn/orchestrate.go`.

## Anti-Patterns

| Don't | Do Instead | Why |
|-------|------------|-----|
| Access ticket files directly | Use SDK client (`internal/cli/sdk/client.go`) | Daemon holds in-memory state with locks |
| Spawn tmux sessions directly | Use `SpawnSession()` via SDK/API | Bypasses session tracking and MCP binding |
| Import `internal/ticket` in CLI code | Use HTTP API endpoints | Breaks daemon-as-authority architecture |
| Import `internal/core/spawn` in CLI | Call `/tickets/{status}/{id}/spawn` | CLI should not import daemon internals |
| Access `.cortex/sessions.json` directly | Use SessionManager API | Bypasses locking and event notifications |

## Debugging

| Symptom | Check |
|---------|-------|
| Daemon not responding | `cortex daemon status`, verify port 4200, check `~/.cortex/logs/` |
| Ticket not found | Verify `X-Cortex-Project` header matches project path |
| Session won't spawn | Check `cortex kanban` for state, use `mode=resume` for orphaned |
| MCP tools not working | Verify `CORTEX_PROJECT_PATH` env, check daemon logs |

## Key Paths

| Component | Path |
|-----------|------|
| CLI commands | `cmd/cortex/commands/` |
| Daemon commands | `cmd/cortexd/commands/` |
| HTTP API handlers | `internal/daemon/api/` |
| MCP tools | `internal/daemon/mcp/` |
| Ticket store | `internal/ticket/` |
| Conclusion store | `internal/conclusion/` |
| Notes store | `internal/notes/` |
| SDK client | `internal/cli/sdk/client.go` |
| Spawn orchestration | `internal/core/spawn/` |
| Project config | `internal/project/config/` |
| Daemon config | `internal/daemon/config/` |
| Tmux manager | `internal/tmux/` |
| TUI components | `internal/cli/tui/` (`views/` wrapper, `kanban/`, `sessions/`, `notes/`, `config/`, `ticket/`) |
| Install/init logic | `internal/install/` |
| Agent defaults | `internal/install/defaults/main/` (shared prompts for all agents) |
| Shared storage | `internal/storage/` |
| Session store | `internal/session/` |
| Response types | `internal/types/` |

## Configuration

**Project** (`cortex.yaml` or `.cortex/cortex.yaml`): Agent type (`claude`, `opencode`) and args per role. Optional `repos` list and `tickets.path`. See `internal/project/config/config.go` for schema.

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
  path: custom/tickets  # optional, defaults to {projectRoot}/tickets
```

**Global** (`~/.cortex/settings.yaml`): Daemon port, bind address (default `127.0.0.1`), log level, project registry. See `internal/daemon/config/config.go` for schema.

**Project registry**: Global config tracks all projects (`projects` list). Auto-registered on `cortex init`. Used by `GET /projects` endpoint.

## CLI Commands

| Command | Description |
|---------|-------------|
| `cortex init` | Initialize `.cortex/` in current directory, register in global config (`--agent claude\|opencode`) |
| `cortex architect` | Start/attach architect session (`--mode fresh\|resume` for orphaned) |
| `cortex project` | Project TUI with tab switching (`tab`/`[`/`]`) |
| `cortex ticket <id>` | Ticket detail TUI |
| `cortex daemon status` | Check daemon status |
| `cortex upgrade` | Refresh `~/.cortex/defaults/` with latest embedded defaults |
| `cortex eject <path>` | Customize a default prompt |

## API Endpoints

Routes defined in `internal/daemon/api/server.go`. SDK client in `internal/cli/sdk/client.go`.

**Global** (no project header): `GET /health`, `GET /projects`, `POST /projects`, global config (`/config/global`), daemon logs/status (`/daemon/logs`, `/daemon/status`).

**Project-scoped** (requires `X-Cortex-Project`): Ticket CRUD, spawn, move, conclude, architect spawn/conclude, session kill/approve, SSE events, conclusions (`/conclusions`), notes CRUD (`/notes`), tags aggregation, project config (`/config/project`, `/config/project/edit`), prompts (`/prompts`, `/prompts/resolve`, `/prompts/eject`, `/prompts/edit`, `/prompts/reset`).

## MCP Tools

Defined in `internal/daemon/mcp/`. Two session types with different tool access:

**Architect** (`tools_architect.go`):

| Tool | Description |
|------|-------------|
| `listProjects` | List all registered projects (for cross-project operations) |
| `listTickets` | List tickets by status (backlog/progress/done), optional search query, tag, and due_before filter |
| `readTicket` | Read full ticket details by ID |
| `createTicket` | Create ticket with title, body, type, repo, optional due_date, references, and tags |
| `updateTicket` | Update ticket title, body, type, references, and/or tags |
| `deleteTicket` | Delete ticket by ID (current project only) |
| `moveTicket` | Move ticket to different status |
| `updateDueDate` | Set or update ticket due date |
| `clearDueDate` | Remove due date from ticket |
| `spawnSession` | Spawn agent session for ticket (modes: normal, resume, fresh) |
| `listNotes` | List all project notes/reminders |
| `createNote` | Create a note with optional due date (YYYY-MM-DD) |
| `updateNote` | Update a note's text and/or due date |
| `deleteNote` | Delete a note by ID |
| `listSessions` | List persistent conclusions (session records) |
| `readSession` | Read a conclusion by ID |
| `concludeSession` | Conclude the architect session and clean up |

**Cross-project support**: Most architect tools accept an optional `project_path` parameter to operate on a different registered project. Use `listProjects` to discover available projects. Exception: `deleteTicket` is restricted to the current project for safety.

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
