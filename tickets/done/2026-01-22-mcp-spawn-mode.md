# MCP Spawn Mode Support

Add mode parameter to MCP spawnSession tool for resume/fresh support.

## Context

This is a fresh project with no users. No backward compatibility needed. Breaking changes are fine. Do not accumulate tech debt.

## Dependencies

Requires `2026-01-22-architect-session-state.md` to be completed first.

## Changes to spawnSession

### Input
```go
type SpawnSessionInput struct {
    TicketID string `json:"ticket_id"`
    Agent    string `json:"agent,omitempty"`  // default: claude
    Mode     string `json:"mode,omitempty"`   // normal (default), resume, fresh
}
```

### Behavior

| State | mode=normal | mode=resume | mode=fresh |
|-------|-------------|-------------|------------|
| Normal | spawn new | error | error |
| Active | error (session in progress) | error | error |
| Orphaned | error (must specify mode) | resume session | clear & spawn new |
| Ended | spawn new | error | spawn new |

### Output
Add state info to output:
```go
type SpawnSessionOutput struct {
    Success    bool
    TicketID   string
    SessionID  string
    TmuxWindow string
    State      string  // the detected state before action
    Message    string
}
```

## Optional: Architect Spawn Tool

Consider adding `spawnArchitect` tool so architect can respawn itself after crash:
```go
type SpawnArchitectInput struct {
    Mode string `json:"mode,omitempty"` // normal, resume, fresh
}
```

This allows the architect to recover from orphaned state.

## Verification

```bash
make lint
make test
make build
make test-integration
```

## Implementation

### Commits Pushed
- `2ccf2ed` feat: add mode parameter to spawnSession for orphaned session handling
- `7a5b113` Merge branch 'ticket/2026-01-22-mcp-spawn-mode'

### Key Files Changed
- `internal/daemon/mcp/types.go` - Added Mode to SpawnSessionInput, State to SpawnSessionOutput
- `internal/daemon/mcp/errors.go` - Added ErrorCodeStateConflict and NewStateConflictError
- `internal/daemon/mcp/tools_architect.go` - Implemented mode-aware handler with state/mode matrix
- `internal/daemon/mcp/tools_test.go` - Added 13 tests covering all state/mode combinations
- `internal/tmux/mock_runner.go` - Added SetWindowExists method for orphaned state testing

### Important Decisions
- Mode defaults to "normal" when not specified (backward compatible)
- STATE_CONFLICT error includes both state and mode in message for clear diagnostics
- State is always returned in output, even on error, for transparency

### Scope Changes
- Deferred spawnArchitect tool to separate task (has additional complexity around self-spawning semantics)
