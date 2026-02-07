---
id: 63e9ca04-0f75-424d-a271-7138eabbbaa0
title: SSE Event System for Real-Time TUI Updates
type: ""
created: 2026-01-26T15:53:01.717454Z
updated: 2026-01-26T18:12:50.163297Z
---
## Context

This project is in early development. Breaking changes are fine. Do not accumulate tech debt — write clean, direct code without backwards-compatibility shims or unnecessary abstractions.

## Problem

The TUI only fetches data on init, manual refresh (`r`), or after user actions. When an agent moves a ticket to review, adds a comment, or concludes a session, the TUI has no way to know until the user manually refreshes.

## Solution

Implement a Server-Sent Events (SSE) system so the daemon can push real-time events to the TUI.

## Architecture

### 1. Event Bus (`internal/events/bus.go`)

Lightweight in-process pub/sub with per-project channels:

```go
type Event struct {
    Type        string // e.g. "ticket.created", "session.status"
    ProjectPath string
    TicketID    string
    Payload     any
}

type Bus struct {
    mu          sync.RWMutex
    subscribers map[string][]chan Event
}

func (b *Bus) Emit(e Event)
func (b *Bus) Subscribe(projectPath string) (<-chan Event, func())
```

### 2. Store-Level Emission

Inject `*events.Bus` into `*ticket.Store`. Emit after successful `save()` in each mutation method:

| Event | Store Method |
|-------|-------------|
| `ticket.created` | `Create()` |
| `ticket.updated` | `Update()` |
| `ticket.deleted` | `Delete()` |
| `ticket.moved` | `Move()` |
| `session.started` | `SetSession()` |
| `session.ended` | `EndSession()` |
| `session.status` | `UpdateSessionStatus()` |
| `comment.added` | `AddComment()` |
| `review.requested` | `AddReviewRequest()` |

### 3. SSE Endpoint (`GET /events`)

New handler in `internal/daemon/api/`:
- Subscribes to event bus for the project (via `X-Cortex-Project` header)
- Writes `text/event-stream` headers
- Streams events as `data: {...}\n\n`
- Cleans up on client disconnect
- Server already has `WriteTimeout: 0` for SSE support

### 4. SDK Client

Add `SubscribeEvents(ctx context.Context) (<-chan Event, error)` to SDK client:
- Opens long-lived HTTP connection to `/events`
- Parses SSE stream
- Returns typed event channel

### 5. TUI Integration

Bubble Tea subscription goroutine reads from event channel, sends `EventMsg` into the program. `Update()` handler re-fetches board data on relevant events.

## Acceptance Criteria

- [ ] Event bus with per-project subscribe/emit
- [ ] Store emits events on all 9 mutation points
- [ ] SSE endpoint streams events to connected clients
- [ ] SDK client can subscribe to event stream
- [ ] TUI auto-refreshes kanban board on incoming events
- [ ] TUI auto-refreshes ticket detail view on incoming events
- [ ] Clean disconnect handling (no goroutine leaks)