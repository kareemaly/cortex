---
id: 6bc3ba94-0281-4abc-ae66-fa3aa5f076c9
author: claude
type: review_requested
created: 2026-02-07T11:22:23.671161Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
        commit: 97dc443
---
## Change

Modified `internal/core/spawn/spawn.go` — the `writeSection` closure in `buildArchitectPrompt()` now appends `(due: YYYY-MM-DD)` when a ticket has a due date set.

**Before**: `- [abc123] Fix login bug (updated: 2026-02-07)`
**After**: `- [abc123] Fix login bug (due: 2026-02-10) (updated: 2026-02-07)`

Tickets without a due date are unaffected.

## Root Cause

The `Due` field from `TicketSummary` was available but not rendered in the architect context injection, so the architect agent couldn't see or reason about deadlines.

## Verification

- `make build` — compiles successfully
- `make test` — all unit tests pass
- `make lint` — 0 issues