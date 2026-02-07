---
id: 2414eb46-2c2f-458a-b804-8de7da57e2cf
title: 'Ticket TUI: Add SSE Subscription for Real-Time Updates'
type: ""
created: 2026-01-27T15:04:26.277294Z
updated: 2026-01-28T06:03:11.657879Z
---
## Summary

The ticket detail TUI (`cortex show`) has no SSE subscription. It defines a `RefreshMsg` type that the kanban pushes into it when embedded, but when opened standalone it never receives real-time updates. Comments, review requests, status changes, and session updates all require a manual refresh.

## Current Behavior

- `RefreshMsg` exists at `model.go:78` but is only triggered externally by the kanban's SSE handler
- Standalone `cortex show <id>` has no event subscription — ticket data is static after initial load

## Expected Behavior

- Ticket TUI subscribes to the project SSE event stream on init
- Filters for events relevant to the current ticket (by ticket ID)
- Automatically reloads ticket data on: `comment_added`, `review_requested`, `session_status`, `session_started`, `session_ended`, `ticket_updated`, `ticket_moved`
- When embedded in kanban, skip own SSE subscription (kanban already handles it via `RefreshMsg`)

## Implementation

- Follow the same pattern as kanban SSE: `subscribeEvents()` → `sseConnectedMsg` → `waitForEvent()` → `RefreshMsg` loop
- Add a flag (e.g., `embedded bool`) to the ticket model constructor so it skips SSE when hosted inside kanban
- Graceful degradation: if SSE connection fails, continue with manual refresh only

## Files
- `internal/cli/tui/ticket/model.go` — add SSE subscription, event filtering, embedded flag
- `cmd/cortex/commands/show.go` — ensure SDK client with project path is passed through