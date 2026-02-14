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
│  MCP Meta       │──┤──────────────▶│      cortexd         │
└─────────────────┘  │               │  ├─ HTTP API         │
┌─────────────────┐  │               │  ├─ StoreManager     │
│  MCP Architect  │──┤               │  ├─ Tmux management  │
└─────────────────┘  │               │  └─ SSE event bus    │
┌─────────────────┐  │               └──────────────────────┘
│  MCP Ticket     │──┘
└─────────────────┘
```

Three-tier agent hierarchy: **Meta** (global) → **Architect** (project-scoped) → **Ticket Agent** (ticket-scoped).

Project context: `X-Cortex-Project` header (HTTP) or `CORTEX_PROJECT_PATH` env (MCP). Meta sessions are global and don't require project context.

## Storage Format

Tickets and docs use **YAML frontmatter + markdown body** stored as `index.md` within a directory-per-entity:

- **Tickets**: `tickets/{status}/{slug}-{shortid}/index.md`
- **Docs**: `docs/{category}/{slug}-{shortid}/index.md`
- **Comments**: `comment-{shortid}.md` files within the entity directory

Default paths are `{projectRoot}/tickets/` and `{projectRoot}/docs/` (configurable via `tickets.path` and `docs.path` in `.cortex/cortex.yaml`). Sessions are ephemeral and stored in `.cortex/sessions.json`.

## Critical Implementation Notes

- **HTTP-only communication**: All clients (CLI, TUI, MCP) communicate via HTTP to daemon. No direct filesystem access to ticket store.
- **Project context**: Always use `X-Cortex-Project` header (HTTP) or `CORTEX_PROJECT_PATH` env (MCP).
- **StoreManager**: Single source of truth for ticket state. Located in `internal/daemon/api/store_manager.go`.
- **DocsStoreManager**: Manages doc stores per project. Located in `internal/daemon/api/docs_store_manager.go`.
- **SessionManager**: Manages ephemeral session stores per project. Located in `internal/daemon/api/session_manager.go`.
- **MetaSessionManager**: Manages the global meta session at `~/.cortex/meta-session.json`. Located in `internal/daemon/api/meta_session_manager.go`.
- **Spawn state detection**: Three states (normal/active/orphaned) with mode matrix (normal/resume/fresh). See `internal/core/spawn/orchestrate.go`.

## Anti-Patterns

| Don't | Do Instead | Why |
|-------|------------|-----|
| Access ticket/doc files directly | Use SDK client (`internal/cli/sdk/client.go`) | Daemon holds in-memory state with locks |
| Spawn tmux sessions directly | Use `SpawnSession()` via SDK/API | Bypasses session tracking and MCP binding |
| Import `internal/ticket` in CLI code | Use HTTP API endpoints | Breaks daemon-as-authority architecture |
| Import `internal/core/spawn` in CLI | Call `/tickets/{status}/{id}/spawn` | CLI should not import daemon internals |
| Access `.cortex/sessions.json` directly | Use SessionManager API | Bypasses locking and event notifications |

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
| Docs store | `internal/docs/` |
| SDK client | `internal/cli/sdk/client.go` |
| Spawn orchestration | `internal/core/spawn/` |
| Project config | `internal/project/config/` |
| Daemon config | `internal/daemon/config/` |
| Tmux manager | `internal/tmux/` |
| Worktree manager | `internal/worktree/` |
| TUI components | `internal/cli/tui/` (`views/` wrapper, `kanban/`, `docs/`, `config/`, `ticket/`) |
| Install/init logic | `internal/install/` |
| Agent defaults | `internal/install/defaults/main/` (shared prompts for all agents) |
| Shared storage | `internal/storage/` |
| Session store | `internal/session/` |
| Response types | `internal/types/` |

## Configuration

**Project** (`.cortex/cortex.yaml`): Self-contained config with agent type (`claude`, `opencode`), agent args, git worktrees, lifecycle hooks, `tickets.path`, `docs.path`. The `extend` field points to `~/.cortex/defaults/main` for prompt resolution only (no config merging). Ticket and doc paths default to `{projectRoot}/tickets` and `{projectRoot}/docs`. See `internal/project/config/config.go` for schema.

**Global** (`~/.cortex/settings.yaml`): Daemon port, bind address (default `127.0.0.1`), log level, project registry. See `internal/daemon/config/config.go` for schema.

**Project registry**: Global config tracks all projects (`projects` list). Auto-registered on `cortex init`. Used by `GET /projects` endpoint and `cortex projects` CLI.

## CLI Commands

| Command | Description |
|---------|-------------|
| `cortex init` | Initialize `.cortex/` in current directory, register in global config (`--agent claude\|opencode`) |
| `cortex meta` | Start/attach global meta session (`--mode fresh\|resume` for orphaned) |
| `cortex architect` | Start/attach architect session (`--mode fresh\|resume` for orphaned) |
| `cortex project` | Project TUI with tab switching (`tab`/`[`/`]`) |
| `cortex show [id]` | Ticket detail TUI |
| `cortex ticket list` | List tickets |
| `cortex ticket spawn <id>` | Spawn ticket agent session |
| `cortex projects` | List all registered projects with ticket counts |
| `cortex register [path]` | Register project in global config |
| `cortex unregister [path]` | Remove project from global config |
| `cortex daemon status` | Check daemon status |
| `cortex upgrade` | Refresh `~/.cortex/defaults/` with latest embedded defaults |
| `cortex eject <path>` | Customize a default prompt |

## API Endpoints

Routes defined in `internal/daemon/api/server.go`. SDK client in `internal/cli/sdk/client.go`.

**Global** (no project header): `GET /health`, `GET /projects`, `POST /projects`, meta session (`/meta/*`), global config (`/config/global`), daemon logs/status (`/daemon/logs`, `/daemon/status`).

**Project-scoped** (requires `X-Cortex-Project`): Ticket CRUD, spawn, move, comments, reviews, conclude, architect spawn/conclude, session kill/approve, SSE events, docs CRUD, tags aggregation, project config (`/config/project`, `/config/project/edit`), prompts (`/prompts`, `/prompts/resolve`, `/prompts/eject`, `/prompts/edit`, `/prompts/reset`).

## MCP Tools

Defined in `internal/daemon/mcp/`. Three session types with different tool access:

**Meta** (`tools_meta.go`) — global session, no project context required:

| Tool | Description |
|------|-------------|
| `listProjects` | List all registered projects |
| `registerProject` | Register a project directory |
| `unregisterProject` | Remove project from registry |
| `spawnArchitect` | Spawn architect for a project |
| `listSessions` | List active sessions for a project (requires project_path) |
| `readProjectConfig` | Read project's cortex.yaml |
| `updateProjectConfig` | Update project's cortex.yaml |
| `readGlobalConfig` | Read global settings.yaml |
| `updateGlobalConfig` | Update global settings.yaml |
| `readPrompt` | Read a prompt template |
| `updatePrompt` | Update/eject a prompt template |
| `readDaemonLogs` | Read recent daemon logs |
| `daemonStatus` | Get daemon uptime, version, project count |
| `concludeSession` | Conclude the meta session |

**Architect** (`tools_architect.go`):

| Tool | Description |
|------|-------------|
| `listProjects` | List all registered projects (for cross-project operations) |
| `listTickets` | List tickets by status (backlog/progress/review/done), optional search query, tag, and due_before filter |
| `readTicket` | Read full ticket details by ID |
| `createTicket` | Create ticket with title, body, type, optional due_date, references, and tags |
| `updateTicket` | Update ticket title, body, references, and/or tags |
| `deleteTicket` | Delete ticket by ID (current project only) |
| `moveTicket` | Move ticket to different status |
| `updateDueDate` | Set or update ticket due date |
| `clearDueDate` | Remove due date from ticket |
| `addTicketComment` | Add comment to ticket (types: review_requested, done, blocker, comment) |
| `spawnSession` | Spawn agent session for ticket (modes: normal, resume, fresh) |
| `createDoc` | Create a markdown doc with frontmatter in a category subdirectory |
| `readDoc` | Read a doc by ID |
| `updateDoc` | Update doc title, body, tags, or references |
| `deleteDoc` | Delete a doc by ID (current project only) |
| `moveDoc` | Move a doc to a different category |
| `listDocs` | List docs with optional category, tag, and search filters |
| `addDocComment` | Add a comment to a documentation file |
| `listSessions` | List all active agent sessions |
| `concludeSession` | Conclude the architect session and clean up |

**Cross-project support**: Most architect tools accept an optional `project_path` parameter to operate on a different registered project. Use `listProjects` to discover available projects. Exception: `deleteTicket` is restricted to the current project for safety.

**Ticket Agent** (`tools_ticket.go`):

| Tool | Description |
|------|-------------|
| `readReference` | Read a referenced ticket or doc by ID |
| `addComment` | Add comment to assigned ticket |
| `addBlocker` | Report blocker on assigned ticket |
| `requestReview` | Request human review, moves ticket to review status |
| `concludeSession` | Complete work, move to done, trigger cleanup |
| `createDoc` | Create a doc for research findings (research type only) |

## Agent Workflow

1. **Meta** (optional) manages projects, configs, and spawns architects across projects
2. Architect reads backlog → calls `spawnSession` for a ticket
3. Daemon creates tmux window with ticket-scoped MCP (30% agent pane, 70% companion pane)
4. Ticket agent works, uses `addComment` to log progress
5. Agent calls `requestReview` when done → ticket moves to review
6. Architect reviews and approves → triggers lifecycle hooks, moves to done

Spawn orchestration handles state detection (normal/active/orphaned) and mode selection (normal/resume/fresh). See `internal/core/spawn/orchestrate.go`. Architect and ticket agent sessions are tracked in `.cortex/sessions.json`; meta sessions are global at `~/.cortex/meta-session.json`. All session types share the same state detection and orphan recovery patterns.

## Lifecycle Hooks

Defined in `.cortex/cortex.yaml` under `lifecycle`. Hooks run on pickup, review, and approve. Template variables: `{{.Slug}}`, `{{.CommitMessage}}`, etc. See `internal/lifecycle/` for execution logic.

## Testing

- Unit: `make test`
- Integration: `make test-integration`
- API tests: `internal/daemon/api/integration_test.go`
- MCP tests: `internal/daemon/mcp/server_test.go`
- Config tests: `internal/daemon/config/config_test.go`
