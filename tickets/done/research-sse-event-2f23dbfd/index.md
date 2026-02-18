---
id: 2f23dbfd-61aa-4ada-980f-2757cfca7a24
title: 'Research: SSE event reliability for TUI live updates'
type: research
tags:
    - research
    - tui
    - api
    - debug
created: 2026-02-15T10:30:45.70557Z
updated: 2026-02-15T11:24:53.669584Z
---
## Problem

SSE (Server-Sent Events) are unreliable across the TUI views:

- **Project TUI (kanban)**: Ticket updates often don't trigger live refreshes — the board doesn't reflect status changes or new tickets without manual intervention.
- **Dashboard**: Sometimes requires pressing `r` to manually refresh, suggesting SSE events are either not arriving or not being handled.

The expectation is that all ticket mutations (create, update, move, delete, comment) should push SSE events that the TUI clients consume to refresh their views in real time.

## Research Goals

1. **Daemon-side event emission**: Trace which operations actually emit SSE events and which don't. Are there gaps where mutations happen without broadcasting an event?
2. **SSE transport layer**: How is the SSE connection managed? Is there reconnection logic if the connection drops? Are events buffered or lost on disconnect?
3. **TUI-side event consumption**: How do the kanban and dashboard TUI views subscribe to and handle SSE events? Do they filter events correctly? Is there error handling for dropped connections?
4. **Race conditions / timing**: Are there scenarios where the SSE event fires before the state is committed, causing stale reads on refresh?
5. **Cross-session events**: When a ticket agent in a different tmux window makes changes, does the event reach the architect's TUI?

## Deliverable

A findings doc summarizing:
- The current SSE architecture (emit → transport → consume)
- Identified gaps or failure modes
- Recommendations for making it reliable