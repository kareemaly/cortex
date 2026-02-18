---
id: b9c946b4-6ca8-40b3-b3aa-49e249b0c1c4
author: claude
type: done
created: 2026-02-13T13:12:57.187605Z
---
## Summary

Fixed the `SessionStatus` SSE event never being emitted.

### Problem
The `SessionStatus` event type was defined in `internal/events/bus.go` but never emitted anywhere. The `POST /agent/status` handler in `agent.go` persisted status changes but didn't notify SSE subscribers.

### Change
**File changed:** `internal/daemon/api/agent.go`

Added `events` import and two `Bus.Emit()` calls — one after each successful `sessStore.UpdateStatus()`:
- Architect session path: emits with `TicketID: "architect"`
- Ticket session path: emits with `TicketID: req.TicketID`

Both include `status` and `tool` in the payload, enabling SSE clients to receive real-time agent status updates.

### Verification
- `make build` — clean
- `make lint` — 0 issues
- `make test` — all pass
- Pre-push hooks passed

### Commit
`f0a87cb` — `fix: emit SessionStatus SSE event on agent status updates`