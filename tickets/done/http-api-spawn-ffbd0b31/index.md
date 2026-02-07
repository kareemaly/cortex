---
id: ffbd0b31-2648-4683-8e97-b04c05115e94
title: HTTP API Spawn Endpoints
type: ""
created: 2026-01-23T06:42:25Z
updated: 2026-01-23T06:42:25Z
---
Add spawn endpoints for architect and update ticket spawn to support mode parameter.

## Context

This is a fresh project with no users. No backward compatibility needed. Breaking changes are fine. Do not accumulate tech debt.

## Dependencies

Requires `2026-01-22-architect-session-state.md` to be completed first.

## Endpoints

### Architect

```
POST /architect/spawn
  Query params: mode=normal|resume|fresh (default: normal)
  Response: { session_id, tmux_session, tmux_window, state }

GET /architect
  Response: { session?, state } (state: normal/active/orphaned/ended)
```

### Ticket Spawn (update existing)

```
POST /tickets/{status}/{id}/spawn
  Query params: mode=normal|resume|fresh (default: normal)
  Response: { session, ticket, state }
```

## Behavior

| State | mode=normal | mode=resume | mode=fresh |
|-------|-------------|-------------|------------|
| Normal | spawn new | error | error |
| Active | return existing, focus window | error | error |
| Orphaned | error | resume session | clear & spawn new |
| Ended | spawn new | error | spawn new |

## Implementation

- Use `internal/core/spawn/` package for all spawn operations
- Use `internal/project/architect/` for architect state
- CLI will call these endpoints via SDK

## Verification

```bash
make lint
make test
make build
make test-integration
```

## Implementation

### Commits

- `ed2b9dc` feat: add HTTP API architect spawn endpoints with mode parameter support

### Key Files Changed

- **`internal/daemon/api/architect.go`** (new) - `ArchitectHandlers` with `GetState` and `Spawn` handlers
- **`internal/daemon/api/types.go`** - Added `ArchitectStateResponse`, `ArchitectSessionResponse`, `ArchitectSpawnResponse`
- **`internal/daemon/api/server.go`** - Added `/architect` route group
- **`internal/daemon/api/tickets.go`** - Updated `Spawn` handler with mode parameter and state detection; removed unused `writeMCPConfig` and `buildAgentCommand` functions

### Decisions

- For Active+normal mode, the handler returns 200 OK (not 201) with the existing session and focuses the tmux window
- Ticket spawn now uses the `spawn.Spawner` from `internal/core/spawn/` for consistency with architect spawn
- Removed legacy MCP config generation code from tickets.go since spawner handles this internally