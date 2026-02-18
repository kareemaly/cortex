---
id: e425a67a-7205-413a-91f7-c6243f82ce54
title: Add SSE reconnection with backoff and periodic poll fallback in TUI
type: work
tags:
    - sse
    - tui
    - reliability
references:
    - doc:0b5f4959-1f8b-46db-b633-88a441d9d305
created: 2026-02-15T11:25:59.261579Z
updated: 2026-02-15T11:39:56.217215Z
---
## Problem

All four TUI views (kanban, ticket, docs, dashboard) have no SSE reconnection logic. When the SSE connection drops for any reason (daemon restart, network hiccup, HTTP timeout), the TUI silently degrades to no live updates permanently. Users must press `r` to manually refresh.

## Requirements

### 1. SSE reconnection with exponential backoff

- When `waitForEvent()` detects a closed channel, return a typed disconnect message (e.g., `sseDisconnectedMsg`) instead of `nil`
- The TUI Update handler should catch this message and attempt to resubscribe after a delay
- Use exponential backoff starting at ~2s, capped at ~30s
- On successful reconnect, reset backoff and do a full data reload
- Apply this pattern to all four TUI views: kanban, ticket, docs, dashboard

### 2. Periodic poll fallback

- Add a safety-net ticker (e.g., every 30-60s) that triggers a data reload regardless of SSE state
- This catches any events that were missed due to drops, buffer overflow, or reconnect gaps
- The poll should be lightweight — it's just calling the existing `loadTickets()` / `loadDocs()` etc.

### 3. Initial connection failure handling

- If the initial SSE connection fails, retry with backoff instead of silently running without live updates

## Acceptance Criteria

- TUI automatically reconnects SSE after daemon restart without user intervention
- A periodic poll ensures the TUI self-heals even if SSE events are missed
- All four TUI views implement the same reconnection pattern
- No regression in normal SSE operation
- Existing tests pass

## References

See findings doc "SSE Event Reliability: Findings & Recommendations" for current code patterns and proposed approach.