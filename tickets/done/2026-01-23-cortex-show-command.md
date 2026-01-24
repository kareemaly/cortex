# Cortex Show Command

## Problem

`cortex ticket <id>` doesn't work - shows help instead of ticket details.

## Expected Behavior

1. Add `cortex show <id>` command to show ticket details
2. Inject `CORTEX_TICKET_ID` env var in tmux windows when spawning ticket sessions
3. `cortex show` (no id) should use `CORTEX_TICKET_ID` env var if available

## Verification

```bash
make lint
make test
make build
make test-integration
```

Test:
1. `cortex show <ticket-id>` should show ticket TUI
2. Inside spawned ticket session: `cortex show` should show current ticket

## Implementation

### Commits Pushed

- `265e622` feat: add cortex show command with CORTEX_TICKET_ID auto-detect

### Key Files Changed

| File | Change |
|------|--------|
| `cmd/cortex/commands/show.go` | **NEW** - Root-level show command that accepts optional ticket ID, falls back to CORTEX_TICKET_ID env var |
| `internal/core/spawn/spawn.go` | Inject CORTEX_TICKET_ID env var when spawning ticket agent sessions |

### Important Decisions

- Reused existing `printTicketPlainText` helper from `ticket_show.go` to avoid code duplication
- Environment variable injection done at command prefix level (`CORTEX_TICKET_ID=<id> <cmd>`) rather than modifying tmux manager interface
- Command registered directly on rootCmd (not as subcommand of ticket)

### Scope Changes

None - implemented exactly as specified in the plan.
