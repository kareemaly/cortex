---
id: 0b5f4959-1f8b-46db-b633-88a441d9d305
title: 'SSE Event Reliability: Findings & Recommendations'
tags:
    - sse
    - tui
    - reliability
    - events
    - real-time
created: 2026-02-15T10:37:59.136873Z
updated: 2026-02-15T10:37:59.136873Z
---
## Overview

This document traces the full SSE event lifecycle — emission → transport → consumption — and identifies gaps, failure modes, and recommendations for making TUI live updates reliable.

---

## Architecture Summary

```
Store Operations (ticket/docs/agent)
         │
         ▼
   Event Bus (in-process, per-project)
   chan Event, capacity 64, non-blocking emit
         │
         ▼
   SSE HTTP Handler (GET /events)
   text/event-stream, flush per event
         │
         ▼
   SDK Client (SubscribeEvents)
   bufio.Scanner, chan Event capacity 64
         │
         ▼
   TUI (Bubbletea)
   waitForEvent() → loadTickets() → waitForEvent()
```

**Key files:**
- Event bus: `internal/events/bus.go`
- SSE handler: `internal/daemon/api/events.go`
- Store emission: `internal/ticket/store.go:30-40`, `internal/docs/store.go:30-39`
- Agent status emission: `internal/daemon/api/agent.go:73,119`
- SDK client: `internal/cli/sdk/client.go:987-1037`
- Kanban TUI: `internal/cli/tui/kanban/model.go`
- Dashboard TUI: `internal/cli/tui/dashboard/model.go`
- Ticket detail TUI: `internal/cli/tui/ticket/model.go`
- Docs TUI: `internal/cli/tui/docs/model.go`

---

## Finding 1: Three Event Types Defined But Never Emitted

**Severity: High** — Direct cause of missing TUI updates

Three of the 12 event types are defined as constants in `internal/events/bus.go:8-21` but are never emitted anywhere in the codebase:

| Event Type | Defined | Emitted | Where It Should Emit |
|---|---|---|---|
| `session_started` | ✓ | ✗ | `spawn.go:209,218,223` — after `SessionStore.Create/CreateArchitect/CreateMeta` |
| `session_ended` | ✓ | ✗ | `sessions.go:122` (Kill), `tickets.go:802` (Conclude), `architect.go:278` (ConcludeArchitect) |
| `review_requested` | ✓ | ✗ | `tickets.go` — when `requestReview` adds a comment with type `review_requested` |

**Impact:** The TUI never learns about session lifecycle changes through SSE. When an agent spawns, completes, or requests review in a different tmux window, the kanban board doesn't refresh. The user must press `r` manually.

**Note:** `review_requested` partially works because `AddComment` emits `comment_added`, but there's no distinct event for review transitions. The kanban reloads on `comment_added` but won't show the status transition until the `TicketMoved` event fires (which does happen via `store.Move()`).

**Recommendation:** Emit `SessionStarted` after each `SessionStore.Create*()` call in `spawn.go`, emit `SessionEnded` after each `SessionStore.End()` call in `sessions.go`, `tickets.go` (conclude), and `architect.go` (conclude). Optionally emit `ReviewRequested` for distinct handling.

---

## Finding 2: Non-Blocking Emit Silently Drops Events

**Severity: Medium** — Can cause missed updates under burst mutation scenarios

The event bus uses non-blocking channel sends (`bus.go:59-63`):

```go
select {
case sub.ch <- e:
default:
    // Subscriber buffer full, DROP EVENT silently
}
```

Each subscriber has a 64-element buffered channel. If the TUI is slow to consume (e.g., blocked on an API call for `loadTickets()`), events are silently dropped with no logging, no counter, no notification.

**Scenario:** An architect creates 10 tickets in rapid succession. The kanban TUI receives the first event, starts a `loadTickets()` API call, and while it's in-flight, the remaining 9 `ticket_created` events fill the buffer. If more events come before the API response, they're dropped. Fortunately, the TUI reloads everything on each event, so it self-heals once the next event arrives — but if ALL events during the reload window are dropped, the board becomes stale until manual refresh.

**Recommendation:**
1. Add a metric/log for dropped events (at minimum, a `slog.Warn` in the `default` branch)
2. Consider a "coalescing" approach: instead of per-mutation events, the TUI could also use a periodic poll (e.g., every 30s) as a safety net alongside SSE

---

## Finding 3: No SSE Reconnection in TUI Views

**Severity: High** — Connection drop = permanent loss of live updates

All four TUI views (kanban, ticket, docs, dashboard) handle SSE connection failure identically:

```go
// kanban/model.go:569-578
func (m Model) subscribeEvents() tea.Cmd {
    return func() tea.Msg {
        ctx, cancel := context.WithCancel(context.Background())
        ch, err := m.client.SubscribeEvents(ctx)
        if err != nil {
            cancel()
            return nil // graceful degradation — NO RECONNECT
        }
        return sseConnectedMsg{ch: ch, cancel: cancel}
    }
}
```

And on channel closure:

```go
// kanban/model.go:587-591
func (m Model) waitForEvent() tea.Cmd {
    ...
    _, ok := <-ch
    if !ok {
        return nil // channel closed — NO RECONNECT
    }
    ...
}
```

**Failure modes:**
- Daemon restart: SSE connection drops, TUI goes permanently stale
- Network hiccup (if daemon is remote): Same result
- HTTP proxy timeout: Same result
- Initial connection failure: TUI starts without SSE, never retries

**Impact:** This is likely the primary cause of the reported "dashboard requires pressing `r` to refresh" issue. Once the SSE connection drops (for any reason), there's no recovery path except quitting and restarting the TUI.

**Recommendation:** Add reconnection with exponential backoff:

```go
case EventMsg:
    return m, tea.Batch(m.loadTickets(), m.waitForEvent())

// New: handle connection loss
case sseDisconnectedMsg:
    m.eventCh = nil
    return m, m.reconnectAfterDelay(2 * time.Second)
```

The `waitForEvent()` function should return a typed `sseDisconnectedMsg` instead of `nil` when the channel closes. The model should then attempt to resubscribe after a delay, with exponential backoff up to a cap (e.g., 30s).

---

## Finding 4: Potential Project Path Mismatch

**Severity: Low-Medium** — Subtle but can break event routing entirely

The SSE subscription path and the event emission path can diverge:

1. **SSE handler** (`events.go:28`): Subscribes using the raw `X-Cortex-Project` header value from `GetProjectPath()` — no cleaning applied in the middleware (`middleware.go:57`)
2. **StoreManager** (`store_manager.go:35`): Applies `filepath.Clean()` before looking up or creating stores
3. **Ticket Store** (`store.go:45`): Stores the cleaned `projectPath` and uses it in all `emit()` calls

If a client sends `X-Cortex-Project: /path/to/project/` (trailing slash), the SSE subscribes for `/path/to/project/` but events are emitted for `/path/to/project`. These are different map keys, so the subscriber receives nothing.

**Recommendation:** Apply `filepath.Clean()` in the `ProjectRequired` middleware before storing in context, ensuring all downstream consumers (SSE handler, store manager, etc.) see the same canonical path.

---

## Finding 5: No Event IDs or Resume Capability

**Severity: Low** — Defense-in-depth, not a primary failure mode

The SSE implementation lacks standard SSE features:

1. **No `id:` field** in SSE output — clients can't track their position
2. **No `Last-Event-ID`** support — reconnecting clients can't resume from where they left off
3. **No `retry:` directive** — clients don't know the server's recommended reconnect interval
4. **No event persistence** — events are fire-and-forget, no history buffer

Current wire format:
```
data: {"type":"ticket_created","project_path":"/path","ticket_id":"abc"}\n\n
```

Missing:
```
id: 42
event: ticket_created
retry: 5000
data: {"project_path":"/path","ticket_id":"abc"}\n\n
```

**Impact:** Even with reconnection logic (Finding 3), events during the disconnect window are permanently lost. The TUI's full-reload-on-any-event approach mitigates this (one event triggers a complete refresh), but there's still a window where the board is stale.

**Recommendation:** This is lower priority since the TUI does full reloads anyway. A periodic poll fallback (see Finding 2) is more practical than implementing full event persistence.

---

## Finding 6: Race Condition Analysis — Events vs State Commitment

**Severity: None (by design)** — The current architecture avoids this

A potential concern was that SSE events might fire before state is committed to disk, causing stale reads when the TUI refreshes. Analysis shows this is NOT an issue:

In `ticket/store.go`, every mutation follows the pattern:
```go
func (s *Store) Create(...) {
    mu.Lock()
    defer mu.Unlock()
    
    s.saveTicket(ticket, status)  // 1. Write to disk FIRST
    s.emit(events.TicketCreated)  // 2. Emit event AFTER write
    return ticket, nil
}
```

The emit always happens after the filesystem write succeeds, within the same mutex lock. The subsequent `loadTickets()` API call from the TUI will read the committed state.

**Status:** No action needed.

---

## Finding 7: Cross-Session Event Delivery

**Severity: None (working as designed)**

When a ticket agent in a different tmux window makes changes (e.g., adds a comment, requests review), the event flow works correctly:

1. Agent calls MCP tool → HTTP API → `store.AddComment()` → `bus.Emit(CommentAdded)`
2. The event bus delivers to all subscribers for that project path
3. The architect's kanban TUI (a separate SSE subscriber) receives the event
4. The kanban reloads all tickets

**Verified:** The event bus is a singleton in the daemon process, and all API handlers (both HTTP and MCP-routed) use the same bus instance. Cross-session event delivery works correctly as long as the SSE connection remains alive (see Finding 3).

---

## Summary of Issues by Severity

| # | Finding | Severity | Effort | Impact |
|---|---------|----------|--------|--------|
| 1 | Missing SessionStarted/SessionEnded/ReviewRequested emissions | High | Low | Direct cause of stale TUI |
| 3 | No SSE reconnection in TUI | High | Medium | Connection drop = permanently stale |
| 2 | Silent event drops on buffer full | Medium | Low | Missed updates under burst load |
| 4 | Project path mismatch (Clean vs raw) | Low-Med | Low | Can break event routing entirely |
| 5 | No event IDs or resume | Low | High | Events lost during disconnect |
| 6 | Race conditions | None | — | Already handled correctly |
| 7 | Cross-session delivery | None | — | Works as designed |

---

## Recommended Fix Priority

### Phase 1 (Quick Wins)
1. **Add missing event emissions** — Emit `SessionStarted` and `SessionEnded` at the identified call sites in `spawn.go`, `sessions.go`, `tickets.go`, and `architect.go`
2. **Normalize project path in middleware** — Add `filepath.Clean()` in `ProjectRequired()` before storing in context
3. **Log dropped events** — Add `slog.Warn` in the `default` branch of `bus.Emit()`

### Phase 2 (Reliability)
4. **Add SSE reconnection with backoff** — Return a typed disconnect message from `waitForEvent()` when channel closes, add reconnect-with-backoff logic in the TUI Update handler
5. **Add periodic poll fallback** — A safety-net ticker (e.g., every 30-60s) that calls `loadTickets()` regardless of SSE state, as defense against any missed events

### Phase 3 (Nice-to-Have)
6. **Add SSE `id:` and `retry:` fields** — Incrementing event IDs and server-recommended retry interval
7. **Consider `event:` field** — Use standard SSE event types instead of encoding type in JSON data