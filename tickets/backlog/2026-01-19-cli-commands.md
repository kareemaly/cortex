# CLI Commands

Implement CLI commands that communicate with the daemon HTTP API.

## Context

The CLI has command stubs that print "not implemented yet". The daemon API now has working endpoints for tickets. Wire them together.

See `DESIGN.md` for:
- CLI commands (lines 44-53)
- Expected behavior for each command

Existing code:
- `cmd/cortex/commands/` - Cobra command stubs
- `internal/cli/sdk/client.go` - HTTP client with Health() method

## Requirements

1. **Extend SDK Client** (`internal/cli/sdk/`)
   - `ListTickets(status)` - GET /tickets or /tickets/{status}
   - `GetTicket(id)` - GET /tickets/{status}/{id}
   - `CreateTicket(title, body)` - POST /tickets
   - `SpawnSession(id)` - POST /tickets/{status}/{id}/spawn
   - `KillSession(id)` - DELETE /sessions/{id}

2. **Implement list Command**
   - `cortex list` - show all tickets grouped by status
   - `cortex list --status=backlog` - filter by status
   - Format output nicely (table or simple list)

3. **Implement spawn Command**
   - `cortex spawn <ticket-id>` - spawn agent for ticket
   - Show session info after spawning
   - Error if ticket not found or already has active session

4. **Implement session Command**
   - `cortex session <id>` - show session details
   - Display ticket info, session status, report if any
   - For now, text output (TUI is separate ticket)

5. **Update version Command**
   - Already works, but ensure it shows daemon version too
   - Call health endpoint to get daemon version

## Verification

```bash
make build   # Builds successfully
make test    # Tests pass
make lint    # No lint errors

# Manual test (daemon must be running)
cortexd &
cortex list
cortex spawn <some-ticket-id>
cortex session <session-id>
```

## Notes

- Commands should error gracefully if daemon not running
- Consider adding --json flag for machine-readable output
- Keep output concise for terminal use
- architect, kanban, install commands remain stubs (separate tickets)
