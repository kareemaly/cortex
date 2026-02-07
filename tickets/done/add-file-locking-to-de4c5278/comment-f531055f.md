---
id: f531055f-91af-4f22-a4b5-c94625ed6332
author: claude
type: ticket_done
created: 2026-01-26T16:07:49.508265Z
---
## Summary

Added per-ticket mutex locking and atomic writes to the ticket store to prevent concurrent write corruption, and improved error logging in the HTTP API error handler.

## Problem

Concurrent writes to the same ticket JSON file (e.g., simultaneous `UpdateSessionStatus` calls from agent hooks) could corrupt the file, producing invalid JSON (extra closing braces). The error was also masked by `handleTicketError` which silently swallowed untyped errors as a generic "internal server error" without logging the actual cause.

## Changes Made

### 1. `internal/ticket/store.go`
- Added `locks sync.Map` field to `Store` struct (maps ticket ID → `*sync.Mutex`)
- Added `ticketMu(id string) *sync.Mutex` helper using `LoadOrStore`
- Added per-ticket mutex locking to all 9 write methods: `Create`, `Update`, `Delete`, `Move`, `SetSession`, `EndSession`, `UpdateSessionStatus`, `AddComment`, `AddReviewRequest`
- In `Delete`, added `s.locks.Delete(id)` to clean up the mutex after file removal
- Rewrote `save()` to use atomic writes: `os.CreateTemp` → write → close → `os.Rename` (temp file in same directory guarantees same-filesystem atomic rename)
- Added `.tmp-` prefix filtering in `List()` to skip crash-orphaned temp files

### 2. `internal/daemon/api/errors.go`
- Added `*slog.Logger` parameter to `handleTicketError`
- Added `logger.Error("internal ticket store error", "error", err)` in the default case

### 3. `internal/daemon/api/tickets.go`
- Updated all 13 `handleTicketError` call sites to pass `h.deps.Logger`

### 4. `internal/ticket/store_test.go`
- Added `TestStoreConcurrentUpdates`: 10 goroutines × 10 updates to the same ticket, verifies ticket remains valid JSON
- Added `TestStoreConcurrentAddComments`: 10 goroutines each adding a comment, verifies all 10 comments are present (no lost updates)

## Key Decisions

- **Per-ticket mutex (not global mutex)**: Allows concurrent writes to different tickets without contention, while serializing writes to the same ticket.
- **sync.Map for mutex storage**: Lock-free reads for the common case; `LoadOrStore` is atomic and avoids the need for a separate initialization lock.
- **Atomic writes via temp+rename**: Defense in depth — even if a process crashes mid-write, the target file is never left in a partial state. Temp files use `.tmp-` prefix and are in the same directory to guarantee same-filesystem rename.
- **Read methods not locked**: `Get`, `List`, `ListAll` don't need locking because atomic writes ensure they always see complete files.

## Verification

- `make test` — all tests pass (including 2 new concurrency tests)
- `make lint` — 0 issues
- `make build` — compiles cleanly

## Files Modified

| File | Lines Changed |
|------|--------------|
| `internal/ticket/store.go` | +72 / -7 |
| `internal/ticket/store_test.go` | +89 / -0 |
| `internal/daemon/api/errors.go` | +4 / -2 |
| `internal/daemon/api/tickets.go` | +13 / -13 |