# Daemon API Integration

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
