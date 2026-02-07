---
id: a5d23602-fea4-41fb-906f-73bcced95c82
author: claude
type: ticket_done
created: 2026-01-26T15:45:53.970316Z
---
## Summary

Limited the "Done" section in the architect prompt to display at most 10 tickets, preventing unbounded context window growth as completed tickets accumulate over time.

## Changes Made

**File modified:** `internal/core/spawn/spawn.go`

In the `buildArchitectPrompt()` function (line ~583), added a guard that truncates `tickets.Done` to the first 10 entries before passing it to `writeSection("Done", ...)`. The Done tickets are already sorted by `Updated` descending in the API handler (`internal/daemon/api/tickets.go:52-59`), so truncating to the first 10 yields the 10 most recently updated done tickets.

The Backlog, In Progress, and Review sections remain unchanged and continue to show all entries.

## Key Decisions

- **Truncate rather than filter:** Simple slice truncation (`doneTickets[:10]`) rather than introducing query parameters or pagination, since the upstream sort order already provides the desired ordering.
- **No changes to other sections:** Only Done tickets accumulate unboundedly; backlog/progress/review are naturally bounded by active work.

## Verification

- `make build` — compiles successfully
- `make test` — all unit tests pass
- `make lint` — 0 issues

## Commit

`7966051` — fix: limit done tickets in architect prompt to latest 10