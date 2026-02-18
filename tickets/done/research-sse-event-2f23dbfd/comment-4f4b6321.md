---
id: 4f4b6321-2003-449f-a22c-dff01e2ecc8d
author: claude
type: done
created: 2026-02-15T11:24:53.667993Z
---
## Research Complete: SSE Event Reliability for TUI Live Updates

### Work Done
Traced the full SSE event lifecycle (emission → transport → consumption) across the codebase, analyzing all relevant files in `internal/events/`, `internal/daemon/api/`, `internal/ticket/`, `internal/docs/`, `internal/cli/sdk/`, and `internal/cli/tui/`.

### Key Findings (5 issues, 2 high severity)

1. **Missing event emissions (High)** — `SessionStarted`, `SessionEnded`, `ReviewRequested` defined in `internal/events/bus.go` but never emitted. Session lifecycle changes from `spawn.go`, `sessions.go`, `tickets.go`, and `architect.go` are invisible to SSE subscribers. Direct cause of stale kanban board.

2. **No SSE reconnection (High)** — All 4 TUI views return `nil` on connection drop with no retry logic. Once SSE disconnects, live updates are permanently lost until TUI restart. Root cause of "dashboard requires pressing `r`" issue.

3. **Silent event drops (Medium)** — Non-blocking emit in `bus.go:59-63` silently discards events when 64-element buffer is full. No logging or metrics.

4. **Project path mismatch (Low-Med)** — `StoreManager` applies `filepath.Clean()` but SSE middleware doesn't, causing potential routing failure on trailing slashes.

5. **No event IDs/resume (Low)** — Missing standard SSE features (id, retry, event fields). Mitigated by full-reload pattern.

### Confirmed Working
- Race conditions: Events always emit after disk writes (correct ordering)
- Cross-session delivery: Event bus singleton correctly routes across tmux windows

### Deliverable
Created findings doc "SSE Event Reliability: Findings & Recommendations" with full architecture trace, code references, and 3-phase remediation plan (quick wins → reliability → nice-to-have).

### No Files Modified
Research-only task. No source files were changed.