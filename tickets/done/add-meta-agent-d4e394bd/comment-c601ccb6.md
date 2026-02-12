---
id: c601ccb6-f259-43fc-a543-e106a6b79464
author: claude
type: review_requested
created: 2026-02-09T15:18:28.50131Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/c35900f5-f98b-476a-a6e6-a9cf41df089a
        commit: 58f810b
---
## Summary

Implemented the meta agent — a global session type that operates above project architects for managing the entire Cortex ecosystem.

### New Files (8)
- `cmd/cortex/commands/meta.go` — CLI `cortex meta` command with `--detach` and `--mode` flags
- `internal/daemon/api/meta.go` — Meta session API handlers (GetState, Spawn, Conclude, Focus)
- `internal/daemon/api/meta_session_manager.go` — Global meta session store at `~/.cortex/meta-session.json`
- `internal/daemon/api/config_handlers.go` — Config read/update API handlers (project + global)
- `internal/daemon/api/logs_handler.go` — Daemon logs + status API handlers
- `internal/daemon/mcp/tools_meta.go` — 18 MCP tools for meta sessions
- `internal/install/defaults/claude-code/prompts/meta/SYSTEM.md` — Meta agent system prompt
- `internal/install/defaults/claude-code/prompts/meta/KICKOFF.md` — Meta agent kickoff template

### Modified Files (20)
- **Session store**: Added `SessionTypeMeta`, `MetaSessionKey`, `CreateMeta`/`GetMeta`/`EndMeta` methods
- **Spawn logic**: Added `AgentTypeMeta` handling in all switch cases, `buildMetaPrompt()`, `DetectMetaState()`
- **Spawn config**: Added `IsMeta` field to `MCPConfigParams`, passes `--meta` flag
- **API server**: Added global routes (`/meta/*`, `/config/*`, `/daemon/*`, `POST /projects`)
- **API deps**: Added `MetaSessionManager` to `Dependencies`
- **API projects**: Added `RegisterProjectHandler` for `POST /projects`
- **MCP server**: Added `IsMeta` config, meta session type detection, `registerMetaTools()` dispatch
- **MCP types**: Added all meta input/output types (RegisterProject, SpawnArchitect, configs, prompts, logs, status)
- **MCP command**: Added `--meta` flag
- **SDK client**: Added `SpawnMeta`, `GetMetaState`, `ConcludeMetaSession`, `FocusMeta`, `RegisterProject`
- **Types**: Added `MetaSpawnResponse`, `MetaStateResponse`
- **Prompt system**: Added `MetaPromptPath`, `BaseMetaPromptPath`, `ResolveMetaPrompt`, `MetaKickoffVars`
- **Project config**: Added `Meta RoleConfig` field
- **Default config**: Added `meta:` section to cortex.yaml

### Meta MCP Tools
Project management: listProjects, registerProject, unregisterProject, spawnArchitect, listSessions
Configuration: readProjectConfig, updateProjectConfig, readGlobalConfig, updateGlobalConfig, readPrompt, updatePrompt
Debugging: readDaemonLogs, daemonStatus
Cross-project: listTickets, readTicket, listDocs, readDoc
Session: concludeSession

### Verification
- `make build` — passes
- `make test` — all tests pass
- `make lint` — 0 issues