---
id: de4c5278-6744-407a-8b2c-33189465885d
title: Add File Locking to Ticket Store
type: ""
created: 2026-01-26T15:58:01.456893Z
updated: 2026-01-26T16:07:49.509835Z
---
## Problem

The ticket store (`internal/ticket/store.go`) has no file locking. When multiple processes (e.g., concurrent `UpdateSessionStatus` calls from agent hooks) write to the same ticket JSON file simultaneously, the file can become corrupted.

**Observed failure:** Extra closing braces (`}}}}`) appended to a ticket JSON file, causing `json.Unmarshal` to fail. This surfaced as a 500 error on `GET /tickets` and `POST /agent/status`, persisting across daemon restarts since the corruption is on disk.

The error was also masked by `handleTicketError` (`internal/daemon/api/errors.go:33`) which swallows non-typed errors as a generic `"internal server error"` — the actual unmarshal error message was lost.

## Solution

Add file-level or per-ticket locking to the store to prevent concurrent writes from corrupting ticket data.

Options:
1. **In-process sync.Mutex per ticket ID** — simplest, works since all writes go through the single daemon process
2. **File-level flock** — protects against external writers too, more robust but more complex
3. **Atomic writes** — write to temp file + rename, prevents partial writes but not concurrent read-modify-write races (need locking too)

Recommended: Option 1 (per-ticket mutex) + Option 3 (atomic writes) for defense in depth.

## Secondary Fix

Improve `handleTicketError` to log the actual error message for untyped errors, rather than swallowing it silently. This will make future debugging easier.

## Acceptance Criteria

- [ ] Concurrent writes to the same ticket do not corrupt the JSON file
- [ ] `handleTicketError` logs the actual error for untyped/internal errors
- [ ] Existing tests pass
- [ ] Add a test for concurrent write safety