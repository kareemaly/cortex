---
id: 810c69b7-d5ff-40a5-8f8c-bf75d75d6d7c
title: 'Clean up session model: add explicit type field, stop overloading ticket_id'
type: work
tags:
    - session
    - data-model
    - cleanup
created: 2026-02-07T12:50:54.655129Z
updated: 2026-02-07T13:11:09.641858Z
---
## Problem

The `Session` struct in `internal/session/session.go` uses `ticket_id` for both ticket sessions (real UUID) and architect sessions (literal string `"architect"`). Downstream code discriminates by checking `ticket_id == ArchitectSessionKey`, which is fragile and semantically confusing. In `sessions.json`, the architect entry shows `"ticket_id": "architect"` which is misleading.

## Solution

Add an explicit `type` field to the Session struct and make `ticket_id` omitempty for architect sessions.

### Changes to Session struct (`internal/session/session.go`):

```go
type SessionType string

const (
    SessionTypeArchitect SessionType = "architect"
    SessionTypeTicket    SessionType = "ticket"
)

type Session struct {
    Type          SessionType `json:"type"`
    TicketID      string      `json:"ticket_id,omitempty"`  // Only populated for ticket sessions
    Agent         string      `json:"agent"`
    TmuxWindow    string      `json:"tmux_window"`
    // ... rest unchanged
}
```

### Backward compatibility

On store load, if `type` is empty, compute it from `ticket_id`:
- If `ticket_id == "architect"` → set `type = "architect"`, clear `ticket_id`
- Otherwise → set `type = "ticket"`

This means no migration script needed — old `sessions.json` files are handled on read.

### Downstream changes

Replace all `ticket_id == ArchitectSessionKey` / `shortID == "architect"` checks with `sess.Type == SessionTypeArchitect`. Key locations:

- `internal/session/store.go` — `CreateArchitect()` sets `Type: SessionTypeArchitect` without `TicketID`
- `internal/daemon/api/sessions.go` (~line 60) — session listing type discrimination
- `internal/daemon/api/agent.go` (~line 60) — agent status update discrimination
- `internal/core/spawn/state.go` — architect state detection
- `internal/core/spawn/spawn.go` (~line 296) — env var setup (keep `CORTEX_TICKET_ID=architect` for backward compat with lifecycle hooks, but use `Type` for logic)
- `internal/daemon/api/architect.go` — conclude response

### What NOT to change

- Map keys in sessions.json stay the same (short ID for tickets, "architect" for architect)
- `ArchitectSessionKey` constant stays as the map key — just stop using it as a ticket_id value
- `CORTEX_TICKET_ID` env var behavior stays for lifecycle hook backward compat

## Acceptance criteria

1. `sessions.json` architect entry has `"type": "architect"` and no `ticket_id` field
2. Ticket session entries have `"type": "ticket"` and `"ticket_id": "<uuid>"`
3. Old `sessions.json` files without `type` are handled gracefully on load
4. All `== ArchitectSessionKey` checks on ticket_id are replaced with type checks
5. No behavioral changes — just cleaner data model