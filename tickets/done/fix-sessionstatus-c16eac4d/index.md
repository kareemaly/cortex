---
id: c16eac4d-51a6-4598-86fb-4a797dc85299
title: Fix SessionStatus SSE event never being emitted
type: work
tags:
    - api
    - tui
    - agent
created: 2026-02-13T13:06:33.83984Z
updated: 2026-02-13T13:12:57.189361Z
---
## Problem

The `SessionStatus` event type is defined in the event bus (`internal/events/bus.go`) but is never emitted when agent status updates come in via `POST /agent/status`. This means TUI clients cannot receive real-time status updates over SSE and must poll instead.

## Requirements

- When `POST /agent/status` successfully updates a session's status, emit a `SessionStatus` event on the bus
- The event payload should include the session/ticket ID, new status, and tool name (if present)
- SSE clients subscribed to `/events` should receive these events in real-time
- This applies to both Claude Code and OpenCode agent status updates

## Acceptance Criteria

- `SessionStatus` events are emitted on every successful status update in `agent.go`
- SSE clients receive real-time agent status change notifications
- Existing status update behavior is unchanged (just adding the event emission)
- Build, lint, and tests pass