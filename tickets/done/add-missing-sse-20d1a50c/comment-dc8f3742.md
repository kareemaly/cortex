---
id: dc8f3742-fa21-41ba-a843-35a0b7d9d4a8
author: claude
type: review_requested
created: 2026-02-15T11:34:16.442519Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/ccc1c875-53c4-4e04-a139-74664dd2eb5f
        commit: da18dd4
---
## Changes

**Emit missing SSE events and fix event routing**

Three SSE event types (`SessionStarted`, `SessionEnded`, `ReviewRequested`) were defined in `internal/events/bus.go` but never emitted. This meant the TUI never learned about session lifecycle changes via SSE.

### 1. Normalize project path in middleware (`middleware.go`)
- Apply `filepath.Clean()` to the project path before storing in context, preventing trailing-slash mismatches between SSE subscriptions and event emissions.

### 2. Log dropped events (`bus.go`)
- Added `slog.Warn` in the `default` branch of the non-blocking emit so dropped events are visible in daemon logs.

### 3. Emit `SessionStarted` (2 sites)
- `tickets.go` `Spawn()` — after successful ticket agent spawn
- `architect.go` `spawnArchitectSession()` — after successful architect spawn

### 4. Emit `SessionEnded` (3 sites)
- `sessions.go` `Kill()` — after `sessStore.End()`
- `tickets.go` `Conclude()` — after `sessStore.End()`
- `architect.go` `Conclude()` — after `sessStore.EndArchitect()`

### 5. Emit `ReviewRequested` (1 site)
- `tickets.go` `RequestReview()` — after moving ticket to review status

### 6. Test fixes
- Added `Bus: events.NewBus()` to all test `Dependencies` structs to prevent nil pointer panics (4 test files).

### Verification
- `make build` — passes
- `make lint` — 0 issues
- `make test` — all tests pass