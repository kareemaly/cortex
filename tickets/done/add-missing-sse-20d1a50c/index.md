---
id: 20d1a50c-494c-4bb5-9a23-8808dc6ae491
title: Add missing SSE event emissions and fix event routing
type: work
tags:
    - sse
    - events
    - api
    - tui
references:
    - doc:0b5f4959-1f8b-46db-b633-88a441d9d305
created: 2026-02-15T11:25:49.214204Z
updated: 2026-02-15T11:35:05.052735Z
---
## Problem

Three SSE event types are defined in `internal/events/bus.go` but never emitted anywhere, causing the TUI to miss session lifecycle changes. Additionally, a path normalization gap can silently break event routing.

## Requirements

### 1. Emit missing events

- **`SessionStarted`**: Emit after each `SessionStore.Create*()` call in the spawn flow
- **`SessionEnded`**: Emit after each `SessionStore.End()` call in session kill, ticket conclude, and architect conclude handlers
- **`ReviewRequested`**: Emit when a ticket moves to review status (optional — `comment_added` + `ticket_moved` already partially cover this, but a distinct event is cleaner)

### 2. Normalize project path in middleware

Apply `filepath.Clean()` to the `X-Cortex-Project` header value in the `ProjectRequired` middleware before storing it in context. This ensures SSE subscriptions and event emissions always use the same canonical path.

### 3. Log dropped events

Add a `slog.Warn` in the `default` branch of the non-blocking emit in the event bus, so dropped events are visible in daemon logs.

## Acceptance Criteria

- All 12 event types defined in `bus.go` are emitted by at least one code path
- Project paths are cleaned in middleware so SSE routing works regardless of trailing slashes
- Dropped events produce a warning log line
- Existing tests pass

## References

See findings doc "SSE Event Reliability: Findings & Recommendations" for code locations and full analysis.