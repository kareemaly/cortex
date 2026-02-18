---
id: a7a2987e-49cd-45b8-9d18-d8d907ed8642
author: claude
type: review_requested
created: 2026-02-13T13:12:14.263626Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Fix: Emit SessionStatus SSE event on agent status updates

### Problem
The `SessionStatus` event type was defined in `internal/events/bus.go` but never emitted. When `POST /agent/status` updated a session's status, it persisted the change but didn't notify SSE subscribers, so real-time status updates were invisible to connected clients.

### Change
**File: `internal/daemon/api/agent.go`**

Added `events` import and two `Bus.Emit()` calls — one after each successful `sessStore.UpdateStatus()`:

1. **Architect session path** (line 73-81): Emits `SessionStatus` event with `ticket_id: "architect"`, status, and tool.
2. **Ticket session path** (line 118-126): Emits `SessionStatus` event with the ticket's ID, status, and tool.

Both events include the `projectPath` so SSE consumers scoped to a project receive them, and `TicketID` so consumers can distinguish architect vs ticket agent sources.

### Verification
- `make build` — compiles cleanly
- `make lint` — 0 issues
- `make test` — all unit tests pass