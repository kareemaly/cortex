# Schema: 1-1 Ticket-Session Relationship

Change ticket-session relationship from one-to-many to one-to-one, and add Claude session ID for resume support.

## Schema Changes

### Ticket
```go
// Before
Sessions []Session

// After
Session *Session  // nil if no session
```

### Session
Add field:
```go
ClaudeSessionID string  // UUID we generate and pass to claude --session-id
```

### Store Methods
- Remove `AddSession()`, `EndSession()`
- Add `SetSession(ticketID, session)` - sets or replaces session
- Add `ClearSession(ticketID)` - removes session (sets to nil)
- Add `UpdateSession(ticketID, fn func(*Session))` - modify existing session

## Files Affected

- `~/projects/cortex1/internal/ticket/ticket.go` - schema
- `~/projects/cortex1/internal/ticket/store.go` - methods
- `~/projects/cortex1/internal/daemon/api/` - handlers and types
- `~/projects/cortex1/internal/daemon/mcp/` - tools and types
- `~/projects/cortex1/internal/cli/sdk/` - client types

## Verification

```bash
make test
make lint
```

## Implementation

### Commits Pushed
- `d13ef7a` refactor: change ticket-session schema from one-to-many to one-to-one

### Key Files Changed
- `internal/ticket/ticket.go` - Changed `Sessions []Session` to `Session *Session`, added `ClaudeSessionID` field, renamed `HasActiveSessions()` to `HasActiveSession()`
- `internal/ticket/store.go` - Replaced `AddSession()` with `SetSession()`, simplified `EndSession()` and `UpdateSessionStatus()` to not require sessionID
- `internal/daemon/api/types.go` - Updated response types for singular session
- `internal/daemon/mcp/types.go` - Updated MCP output types for singular session
- `internal/cli/sdk/client.go` - Updated SDK types and `FindSession()` method

### Important Decisions
- Simplified store methods: `SetSession(ticketID, agent, tmuxWindow, claudeSessionID)` replaces any existing session rather than having separate Add/Clear/Update methods
- `EndSession(ticketID)` no longer needs sessionID since there's only one session
- Removed backward compatibility code since this is a fresh project with no users

### Scope Changes
- Original plan included `ClearSession()` and `UpdateSession()` methods - simplified to just `SetSession()` and `EndSession()`
- Did not implement backward-compatible JSON unmarshaling (removed per user feedback)
