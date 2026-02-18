---
id: 84adcbb2-5c05-440f-a6d8-e8268e30ed84
author: claude
type: review_requested
created: 2026-02-15T11:39:01.209802Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/b5865cb9-796d-458a-ad92-8f9b7ec9b8ea
        commit: 27491a2
---
## Summary

Added automatic SSE reconnection with exponential backoff and periodic poll fallback to all four TUI views (kanban, docs, ticket, dashboard).

### Changes across 4 files (323 lines added, 18 removed):

**New message types** (per view):
- `sseDisconnectedMsg` / `SSEDisconnectedMsg` — returned when SSE connection fails or channel closes
- `sseReconnectTickMsg` / `SSEReconnectTickMsg` — timer-based reconnect trigger
- `pollTickMsg` / `PollTickMsg` — 60-second periodic data refresh

**New model fields:**
- `sseBackoff` (duration) / `sseBackoffs` (map for dashboard) — tracks exponential backoff per connection
- `sseConnected` (bool) — stale-disconnect guard for single-connection views

**Behavior:**
1. `subscribeEvents()` now returns `sseDisconnectedMsg` on error instead of `nil`
2. `waitForEvent()` returns `sseDisconnectedMsg` on channel close instead of `nil`
3. `sseConnectedMsg` handler cancels old context, resets backoff, sets connected flag
4. `sseDisconnectedMsg` handler uses stale guard, advances backoff (2s→4s→8s→...→30s cap), schedules reconnect
5. `Init()` adds `startPollTicker()` (60s safety-net refresh)
6. Dashboard: per-project backoff map, guards on `sseChannels[path]` existence, refresh key resets backoff map

**Stale disconnect guard (critical):** When a new SSE connection replaces an old one, the old context is cancelled, closing the old channel. The old `waitForEvent` goroutine returns `sseDisconnectedMsg`. Without the guard, this would trigger a spurious reconnect loop. Single-connection views guard on `sseConnected`; dashboard guards on `sseChannels[projectPath]`.

### Verification
- `make build` — compiles clean
- `make lint` — 0 issues
- `make test` — all unit tests pass