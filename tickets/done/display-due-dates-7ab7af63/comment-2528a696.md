---
id: 2528a696-708e-4247-b4e2-d02066933d61
author: claude
type: done
created: 2026-02-07T11:23:41.647412Z
---
## Summary

Added due date display to the architect context injection in `internal/core/spawn/spawn.go`.

### Change
Modified the `writeSection` closure in `buildArchitectPrompt()` (lines 624-630) to append `(due: YYYY-MM-DD)` when a ticket's `Due` field is set. Tickets without due dates are unaffected.

**Output format**:
- With due date: `- [abc123] Fix login bug (due: 2026-02-10) (updated: 2026-02-07)`
- Without due date: `- [abc123] Fix login bug (updated: 2026-02-07)` (unchanged)

### Files changed
- `internal/core/spawn/spawn.go` — 5 lines added, 1 removed

### Verification
- `make build` — compiles successfully
- `make test` — all unit tests pass
- `make lint` — 0 issues

### Commit
`97dc443` — `feat: display due dates in architect context injection`