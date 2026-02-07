---
id: cd4ea148-3dd5-4fb8-9d87-5ac93fc39600
author: claude
type: review_requested
created: 2026-02-07T11:43:48.429817Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Architect Session Tracking — Full Implementation

Brings architect sessions to parity with ticket agent sessions: persisted in session store, orphan detection with fresh/resume modes, `concludeSession` MCP tool, rich dashboard TUI display, and inclusion in `listSessions`.

### Files Changed (14 files)

**Core session/spawn layer:**
- `internal/session/session.go` — Added `ArchitectSessionKey` constant
- `internal/session/store.go` — Added `CreateArchitect`, `GetArchitect`, `EndArchitect` methods
- `internal/core/spawn/spawn.go` — Extended `SessionStoreInterface`, architect session creation/cleanup/env vars in Spawn/Resume/Fresh
- `internal/core/spawn/spawn_test.go` — Updated mock with new interface methods, fixed `TestResume_NoTicketID`
- `internal/core/spawn/state.go` — Added `DetectArchitectState` (handles pre-migration windows)

**Response types:**
- `internal/types/response.go` — Extended `ArchitectSessionResponse` with `Status`, `Tool`, `IsOrphaned`

**API handlers:**
- `internal/daemon/api/architect.go` — Rewrote `GetState`/`Spawn` with session store + state detection, added `Conclude` handler
- `internal/daemon/api/server.go` — Added `/architect/conclude` route
- `internal/daemon/api/agent.go` — Handle `ticket_id: "architect"` for status updates
- `internal/daemon/api/sessions.go` — Added `session_type` field, handle architect in list

**SDK + MCP:**
- `internal/cli/sdk/client.go` — Added `ConcludeArchitectSession`, `SessionType` to `SessionListItem`
- `internal/daemon/mcp/tools_architect.go` — Registered `concludeSession` tool + handler
- `internal/daemon/mcp/types.go` — Added `SessionType` to `SessionListItem`, `ArchitectConcludeOutput`

**CLI + TUI:**
- `cmd/cortex/commands/architect.go` — Added `--mode` flag, orphaned session error handling
- `internal/cli/tui/dashboard/model.go` — Rich architect status display (icon, tool, duration, orphaned badge)

### Verification
- `make build` — compiles cleanly
- `make test` — all tests pass
- `make lint` — 0 issues