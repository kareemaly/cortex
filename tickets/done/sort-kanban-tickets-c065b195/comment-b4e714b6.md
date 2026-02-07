---
id: b4e714b6-f221-424e-95f5-acdf5b41c236
author: claude
type: ticket_done
created: 2026-01-27T10:11:09.802132Z
---
## Summary

Changed kanban ticket sorting from `updated_at` to `created_at` (descending, newest first) in both API list handlers. This prevents tickets from constantly reordering in the kanban UI while agents are actively working on them, since `created_at` is immutable.

## Changes Made

**File: `internal/daemon/api/tickets.go`** (1 file, 9 insertions, 9 deletions)

1. **`ListAll` handler (lines 55-62)**: Renamed sort function from `sortByUpdated` to `sortByCreated` and changed comparator from `b.Updated.Compare(a.Updated)` to `b.Created.Compare(a.Created)`. Applied to all four columns (Backlog, Progress, Review, Done).

2. **`ListByStatus` handler (lines 95-98)**: Changed inline sort comparator from `b.Updated.Compare(a.Updated)` to `b.Created.Compare(a.Created)`.

## Key Decisions

- **Sort by `created_at` rather than removing sorting entirely**: Maintains a predictable, stable order (newest tickets first) without the churn caused by `updated_at` changing during agent work.
- **No display changes**: The ticket card metadata line continues to show the same date fields — only the sort key changed.

## Verification

- `make build` — compiles successfully
- `make test` — all unit tests pass
- `make lint` — 0 issues

## Follow-up

None. The change is minimal and self-contained.