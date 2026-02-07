---
id: 56c95616-92e6-43af-819e-07d46b6fb474
title: Emit SSE Events on All Ticket Status Changes
type: ""
created: 2026-01-27T10:09:35.156409Z
updated: 2026-01-27T10:37:43.706457Z
---
## Problem

The kanban TUI doesn't update in real-time when tickets move between columns. Specifically:

- **Spawning a session** moves a ticket to progress, but no SSE event is emitted — the TUI only reflects the change after a manual refresh (`r`)
- **Concluding a session** (approve) moves a ticket to done, but likely has the same issue

The TUI has SSE support for real-time updates, but not all daemon API endpoints that change ticket status are emitting events.

## Solution

Audit all daemon API and MCP endpoints that move tickets between statuses and ensure each one emits an SSE event so the TUI updates in real-time.

## Scope

- **Review all daemon API handlers** (`internal/daemon/api/`) and **MCP tool handlers** (`internal/daemon/mcp/`) for any operation that changes ticket status:
  - `moveTicket` — explicit status change
  - `spawnSession` — moves ticket to progress
  - `approve` / session conclude — moves ticket to done
  - `pickupTicket` — moves ticket to progress
  - Any other path that calls ticket status transitions
- Ensure each of these emits an SSE event after the status change
- Verify the TUI's SSE handler picks up these events and refreshes the board

## Acceptance Criteria

- [ ] Spawning a session triggers an SSE event that the TUI receives
- [ ] Concluding/approving a session triggers an SSE event
- [ ] All ticket status transitions across both API and MCP emit SSE events
- [ ] TUI updates in real-time without manual refresh for all status changes