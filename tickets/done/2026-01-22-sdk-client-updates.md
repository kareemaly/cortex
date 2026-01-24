# SDK Client Updates

Update SDK client to support new API endpoints for architect and spawn modes.

## Context

This is a fresh project with no users. No backward compatibility needed. Breaking changes are fine. Do not accumulate tech debt.

## New Methods

### Architect
```go
// Get architect session state
GetArchitect() (*ArchitectStateResponse, error)

// Spawn architect with mode (normal/resume/fresh)
SpawnArchitect(mode string) (*ArchitectSpawnResponse, error)
```

### Ticket Spawn (update existing)
```go
// Update signature to include mode
SpawnSession(status, ticketID, mode string) (*SpawnResponse, error)
```

## Response Types

Add types matching API responses:
- `ArchitectStateResponse` - session info + state
- `ArchitectSpawnResponse` - spawn result with session
- Update `SpawnResponse` to include state

## Remove

- `FindSession()` - no longer needed with 1-1 session relationship

## Used By

- `cmd/cortex/commands/` - CLI commands
- `internal/cli/tui/kanban/` - Kanban TUI

## Verification

```bash
make lint
make test
make build
make test-integration
```

## Implementation

### Commits Pushed
- `5e4f35e` feat: add architect endpoints and spawn mode to SDK client

### Key Files Changed
- `internal/cli/sdk/client.go` - Added architect types and methods, updated SpawnSession signature, removed FindSession
- `cmd/cortex/commands/spawn.go` - Updated SpawnSession call to pass mode parameter
- `cmd/cortex/commands/session.go` - Changed from session ID to ticket ID lookup
- `internal/cli/tui/kanban/model.go` - Updated spawnSession call, implemented openArchitect

### Changes Made
1. Added three new response types: `ArchitectSessionResponse`, `ArchitectStateResponse`, `ArchitectSpawnResponse`
2. Added `GetArchitect()` method for GET `/architect`
3. Added `SpawnArchitect(mode string)` method for POST `/architect/spawn`
4. Updated `SpawnSession(status, id, mode string)` to accept mode parameter and append to URL
5. Removed `FindSession()` method (no longer needed with 1-1 session relationship)
6. Updated `session` command to use `cortex session <ticket-id>` instead of `cortex session <session-id>`
7. Implemented kanban `openArchitect()` to call the new `SpawnArchitect` endpoint

### Scope Changes
- None, implemented as specified in the plan
