---
id: 7cb887be-36e1-4bfd-83c3-9cd0adee76af
title: Daemon API Integration
type: ""
created: 2026-01-20T14:14:33Z
updated: 2026-01-20T14:14:33Z
---
Wire project config, lifecycle hooks, and session spawning into daemon HTTP API.

## Context

The daemon API has ticket endpoints but spawn is stubbed and hooks aren't executed. The supporting packages are ready for integration.

See `DESIGN.md` for:
- Daemon API endpoints (lines 399-418)
- Spawn endpoint behavior (line 411)
- Session endpoints (lines 415-416)

Existing packages to integrate:
- `internal/lifecycle` - hook execution
- `internal/tmux` - session management
- `internal/project/config` - load config and hooks
- `internal/git` - get git_base for sessions

## Requirements

Update `internal/daemon/api/` and `cmd/cortexd/`:

1. **Load Project Config at Startup**
   - Detect project root from current directory or flag
   - Load `.cortex/cortex.yaml` using project/config package
   - Pass config to handlers that need it

2. **Wire Spawn Endpoint**
   - `POST /tickets/{status}/{id}/spawn`
   - Create tmux window for ticket
   - Generate MCP config file
   - Run agent command in window
   - Record session with git_base
   - Move ticket to progress
   - Return session info

3. **Wire Kill Session Endpoint**
   - `DELETE /sessions/{id}`
   - Kill tmux window
   - End session in ticket store
   - Return success/failure

4. **Hook Execution on Move** (optional)
   - When moving to progress, consider on_pickup
   - When moving to done, consider on_approve
   - Or leave hooks to MCP tools only

## Verification

```bash
make build   # Builds successfully
make test    # Tests pass
make lint    # No lint errors

# Manual test
cortexd &
curl -X POST http://localhost:4200/tickets/backlog/{id}/spawn
```

## Notes

- Spawn should work from any ticket status (usually backlog)
- Session kill should be idempotent
- Project path could be passed via flag or env var
- Consider adding project path to config response

## Implementation

### Commits

- `889a065` feat: wire project config and tmux into daemon API spawn/kill endpoints

### Key Files Changed

- `internal/daemon/api/deps.go` (new) - Dependencies struct centralizing handler dependencies
- `cmd/cortexd/commands/serve.go` - Load project config, tmux manager, and lifecycle executor at startup
- `internal/daemon/api/server.go` - Accept Dependencies instead of individual stores
- `internal/daemon/api/tickets.go` - Implement Spawn handler with full workflow
- `internal/daemon/api/sessions.go` - Implement Kill handler with session lookup
- `internal/daemon/api/types.go` - Add SpawnResponse type

### Key Decisions

1. **Dependencies Struct**: Created centralized Dependencies struct to avoid passing many parameters to handlers
2. **Graceful Degradation**: TmuxManager is nil if tmux not installed; Spawn returns 503 in this case
3. **Project Config Fallback**: If no `.cortex` directory found, uses DefaultConfig with warning log
4. **MCP Config**: Written to temp file with format `cortex-mcp-*.json`
5. **Session Naming**: Tmux session uses project name (default: "cortex"), window uses slugified ticket title
6. **Kill Idempotency**: Kill succeeds even if tmux window already closed; logs warning but doesn't fail

### Scope Changes

- Deferred lifecycle hook execution (on_pickup) to future work - the executor is initialized but hooks not wired into spawn flow yet