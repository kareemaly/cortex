---
id: d4e394bd-07d8-47f2-b800-e2d2aec89ba7
title: 'Add meta agent: global session for managing projects and Cortex itself'
type: work
tags:
    - meta-agent
    - mcp
    - session
    - api
    - prompts
created: 2026-02-09T14:49:08.66741Z
updated: 2026-02-09T15:20:51.800532Z
---
## Goal

Add a new "meta" agent session type that operates at the global level — above architects. The meta agent manages the entire Cortex ecosystem: registering projects, configuring workflows, customizing prompts, spawning architects, and debugging daemon issues.

## Hierarchy

```
Meta Agent (one per daemon, global scope)
  └─ manages N projects
       └─ each has an Architect (project scope)
            └─ spawns Ticket Agents (ticket scope)
```

## Session Design

- **Launch**: `cortex meta` (new CLI command)
- **Companion pane**: `cortex dashboard`
- **Session tracking**: `~/.cortex/meta-session.json` (global, not per-project)
- **Tmux**: Own window in the daemon's tmux session, same spawn pattern as architect
- **State detection**: Same normal/active/orphaned model with resume/fresh modes
- **One at a time**: Only one meta session can be active globally

## MCP Tool Set (`tools_meta.go`)

### Project Management

| Tool | Description |
|------|-------------|
| `listProjects` | List all registered projects with paths, agent type, ticket counts |
| `registerProject` | Register a new project path in global config |
| `unregisterProject` | Remove a project from global registry |
| `spawnArchitect` | Spawn an architect session for a given project (requires project_path) |
| `listSessions` | List all active sessions across all projects |

### Configuration

| Tool | Description |
|------|-------------|
| `readProjectConfig` | Read a project's `.cortex/cortex.yaml` (requires project_path) |
| `updateProjectConfig` | Update cortex.yaml fields — agent type, agent args, lifecycle hooks, tickets.path, docs.path, worktree settings. Structured input, validates before writing. |
| `readGlobalConfig` | Read `~/.cortex/settings.yaml` (daemon port, bind address, log level, etc.) |
| `updateGlobalConfig` | Update global daemon settings. Structured input, validates before writing. |
| `readPrompt` | Read a prompt file — returns ejected version if exists, otherwise the default. Params: project_path, role (architect/ticket), stage (system/kickoff/approve) |
| `updatePrompt` | Update a prompt file. Auto-ejects from defaults to project `.cortex/prompts/` if not already ejected, then applies the update. Params: project_path, role, stage, content |

### Debugging

| Tool | Description |
|------|-------------|
| `readDaemonLogs` | Read recent daemon logs from `~/.cortex/logs/`. Optional: line count, level filter |
| `daemonStatus` | Health check + daemon info (port, uptime, registered project count) |

### Cross-Project Awareness

| Tool | Description |
|------|-------------|
| `listTickets` | List tickets for a project (requires project_path) |
| `readTicket` | Read a ticket from any project (requires project_path + ticket_id) |
| `listDocs` | List docs from any project (requires project_path) |
| `readDoc` | Read a doc from any project (requires project_path + doc_id) |

## Prompts

### New files

| File | Purpose |
|------|---------|
| `internal/install/defaults/claude-code/prompts/meta/SYSTEM.md` | Meta agent system prompt — role, tool descriptions, workflow guidance |
| `internal/install/defaults/claude-code/prompts/meta/KICKOFF.md` | Kickoff template with dynamic injection |

### System Prompt Content

The meta agent knows about:
- Cortex architecture (daemon ↔ projects ↔ agents)
- File layout conventions (tickets/, docs/, .cortex/)
- cortex.yaml schema and all configurable fields
- Prompt customization workflow (readPrompt → updatePrompt auto-ejects)
- How to debug common issues (daemon logs, session states, port conflicts)
- Global config schema (~/.cortex/settings.yaml)

### Kickoff Injection

`buildMetaPrompt()` in spawn.go — similar pattern to architect but global scope:

```markdown
# Cortex Meta Session

**Current date**: {{.CurrentDate}}

# Registered Projects

{{.ProjectList}}
## project-name (path)
  - agent: claude-code
  - backlog: 3, progress: 1, review: 0, done: 12
  - active sessions: 2

# Active Sessions

{{.SessionList}}
```

Template vars:
```go
type MetaKickoffVars struct {
    CurrentDate string
    ProjectList string
    SessionList string
}
```

## Implementation

### New Files

| File | Purpose |
|------|---------|
| `cmd/cortex/commands/meta.go` | CLI command `cortex meta` with --mode flag |
| `internal/daemon/mcp/tools_meta.go` | MCP tool definitions and handlers |
| `internal/daemon/api/meta.go` | API endpoints for meta session spawn/conclude |
| `internal/install/defaults/claude-code/prompts/meta/SYSTEM.md` | System prompt |
| `internal/install/defaults/claude-code/prompts/meta/KICKOFF.md` | Kickoff template |

### Modified Files

| File | Change |
|------|--------|
| `internal/daemon/api/server.go` | Add meta session routes + config/prompt/logs endpoints |
| `internal/cli/sdk/client.go` | Add SDK methods for meta endpoints |
| `internal/core/spawn/spawn.go` | Add `buildMetaPrompt()`, handle meta agent type in spawn |
| `internal/session/store.go` | Support meta session in global location (`~/.cortex/meta-session.json`) |
| `internal/prompt/template.go` | Add `MetaKickoffVars` struct |
| `internal/daemon/mcp/server.go` | Register meta tool set based on session type |
| `internal/core/spawn/launcher.go` | Handle meta agent launch (companion = dashboard) |

### API Endpoints (new)

| Route | Method | Description |
|-------|--------|-------------|
| `/meta/spawn` | POST | Spawn meta session |
| `/meta/conclude` | POST | Conclude meta session |
| `/config/project` | GET | Read project config (requires X-Cortex-Project) |
| `/config/project` | PUT | Update project config |
| `/config/global` | GET | Read global config |
| `/config/global` | PUT | Update global config |
| `/prompts/read` | GET | Read prompt (ejected or default) |
| `/prompts/update` | PUT | Update prompt (auto-eject + write) |
| `/daemon/logs` | GET | Read daemon logs |
| `/daemon/status` | GET | Extended daemon status |

## Notes

- The `updatePrompt` tool is the key UX win — it auto-ejects the default prompt to the project's `.cortex/prompts/` directory if not already customized, then applies the edit. One tool, no manual eject step.
- `updateProjectConfig` and `updateGlobalConfig` should accept partial updates (merge, not replace) and validate the result before writing.
- Cross-project tools (listTickets, readTicket, etc.) reuse existing API endpoints — just pass project_path as the X-Cortex-Project header.
- Meta session is global so it doesn't have a project context — the SDK client should work without a default project path.
- The meta MCP server needs to be bound without `CORTEX_PROJECT_PATH` since it's global.