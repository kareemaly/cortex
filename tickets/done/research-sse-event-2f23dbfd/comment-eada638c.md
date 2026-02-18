---
id: eada638c-5411-47ac-a39e-49806034c63e
author: claude
type: review_requested
created: 2026-02-15T10:38:15.364992Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/748328e3-0adb-4ea3-87bf-5124c3f55b63
---
## Research Complete: SSE Event Reliability for TUI Live Updates

### Summary

Traced the full SSE event lifecycle across the codebase and identified 5 issues (2 high severity, 1 medium, 2 low).

### Key Findings

**High Severity:**
1. **Three event types defined but never emitted** — `SessionStarted`, `SessionEnded`, and `ReviewRequested` are constants in `internal/events/bus.go` but no code ever calls `bus.Emit()` with them. Session lifecycle changes (spawn, kill, conclude) are invisible to SSE subscribers. This is a direct cause of the kanban not refreshing when agents start/stop.

2. **No SSE reconnection in any TUI view** — When the SSE connection drops (daemon restart, network issue), the TUI silently degrades to no live updates permanently. `waitForEvent()` returns `nil` on channel close with no reconnection attempt. This is the likely root cause of the "dashboard requires pressing `r`" issue.

**Medium Severity:**
3. **Silent event drops on buffer full** — The event bus uses non-blocking sends with a 64-element buffer. Under burst mutations, events are silently discarded. No logging or metrics exist for drops.

**Low Severity:**
4. **Project path mismatch** — `StoreManager.GetStore()` applies `filepath.Clean()` before creating stores (and thus emitting events), but the SSE handler subscribes using the raw header value. A trailing slash in `X-Cortex-Project` would cause complete event routing failure.

5. **No event IDs or resume capability** — Standard SSE features (id, retry, event fields) are missing, though the full-reload-on-any-event TUI pattern mitigates this.

**Confirmed Working:**
- Race conditions: Events always emit after disk writes (correct ordering)
- Cross-session delivery: Event bus singleton correctly routes across tmux windows

### Recommended Fix Priority
- **Phase 1 (Quick Wins):** Add missing emissions, normalize paths in middleware, log drops
- **Phase 2 (Reliability):** SSE reconnection with backoff, periodic poll fallback
- **Phase 3 (Nice-to-Have):** Standard SSE features (id, retry fields)

### Deliverable
Created findings doc: "SSE Event Reliability: Findings & Recommendations" with full architecture trace, code references, and phased remediation plan.