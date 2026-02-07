---
id: 3172a997-8638-4945-a314-ad213b6ae54b
author: claude
type: ticket_done
created: 2026-01-27T10:14:39.942441Z
---
## Summary

Refactored the MCP `handleSpawnSession` in `tools_architect.go` to delegate spawn requests to the daemon's HTTP API (`POST /tickets/{status}/{id}/spawn`) instead of calling `spawn.Orchestrate()` directly. This ensures the daemon is the single owner of spawn logic — updating and restarting the daemon immediately affects all MCP sessions without needing to restart Claude Code.

## Changes Made

### 1. `internal/daemon/mcp/types.go`
- Removed the `Agent` field from `SpawnSessionInput`. The daemon resolves the agent from project config via its existing fallback chain (request → project config → "claude"), so the MCP tool no longer needs to pass it.

### 2. `internal/daemon/mcp/server.go`
- Added default `DaemonURL` (`http://localhost:4200`) for architect sessions in `NewServer()`. This ensures the spawn handler always has a daemon URL to call, matching the daemon's default port.

### 3. `internal/daemon/mcp/tools_architect.go`
- Replaced the entire `handleSpawnSession` implementation:
  - Looks up ticket status via local store to build the correct URL path
  - Makes HTTP POST to `{DaemonURL}/tickets/{status}/{id}/spawn?mode={mode}` with `X-Cortex-Project` header
  - Maps HTTP response codes to MCP output/errors:
    - 201 Created → success with session ID and tmux window
    - 200 OK → StateConflictError("active", ...)
    - 409 Conflict → StateConflictError with state extracted from error code/message
    - 400 Bad Request → non-error failure with message
    - 503 Service Unavailable → non-error failure with message
    - Other → non-error failure with message
  - Added `parseStateFromError()` helper to extract session state from API error codes ("session_orphaned" → "orphaned") and error messages (parsing "in state <state>" format)
- Removed imports: `spawn`, `tmux`, `errors`
- Added imports: `encoding/json`, `fmt`, `net/http`, `api`, `types`

### 4. `internal/daemon/mcp/tools_test.go`
- Replaced `setupTestServerWithMockTmux` and `setupTestServerWithOrphanedSession` with unified `setupArchitectWithDaemon(t, windowExists bool)` that:
  - Creates a temp project directory with `.cortex/tickets`, `.cortex/prompts/ticket-system.md`, and `.cortex/cortex.yaml` (with `name: test-session` for tmux session name)
  - Creates a mock tmux runner (with configurable window existence)
  - Starts an HTTP test server with the full daemon API router
  - Creates an MCP server with `DaemonURL` pointing to the test server
- Updated `TestHandleSpawnSessionActiveSession` to expect `StateConflictError` (previously tested a different code path — the old handler returned a non-error "CORTEX_TMUX_SESSION not configured" message)
- Dropped `State` field assertions from success cases (201 responses don't carry state info)
- All 19 spawn tests pass through the full HTTP delegation path

## Key Decisions

1. **Use `net/http` directly instead of SDK client**: The existing `sdk.SpawnSession` decodes into `SessionResponse` instead of `SpawnResponse`, making it unsuitable. Direct HTTP calls give precise control over response mapping.

2. **Local store for ticket status lookup**: The MCP handler still uses `s.store.Get()` to find the ticket's current status (needed to build the URL path `/tickets/{status}/{id}/spawn`). This is a read-only operation that doesn't need to go through the daemon.

3. **Omit State field from success responses**: When the daemon returns 201 Created, we don't have direct access to the state enum. Since the AI agent only needs to know the spawn succeeded, omitting it simplifies the code without loss of useful information.

4. **State extraction from error messages**: For 409 state_conflict responses, the state is parsed from the error message format ("spawn: ticket <id> in state <state>: <message>"). For session_orphaned, it's derived directly from the error code.

## Files Modified
- `internal/daemon/mcp/types.go` — 1 line removed
- `internal/daemon/mcp/server.go` — 4 lines added
- `internal/daemon/mcp/tools_architect.go` — 149 lines changed (net: added HTTP delegation, removed direct orchestration)
- `internal/daemon/mcp/tools_test.go` — 159 lines changed (net: unified test helpers, updated assertions)

## Verification
- `make build` — compiles successfully
- `make lint` — 0 issues
- `make test` — all tests pass