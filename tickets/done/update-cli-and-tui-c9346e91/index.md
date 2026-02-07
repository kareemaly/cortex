---
id: c9346e91-b1e6-43dd-8ceb-63532741634b
title: Update CLI and TUI for new storage layer
type: work
created: 2026-02-07T09:59:55.476932Z
updated: 2026-02-07T10:13:15.310736Z
---
## Overview

Update all CLI commands and TUI components to work correctly with the new frontmatter + directory-per-entity storage layer. Ticket 2a patched these to compile, but they need proper updates to render new fields, use new paths, and handle the decoupled session model.

## Context

Key changes from the storage migration that affect CLI/TUI:
- **Flat dates**: `Created`, `Updated`, `Due` directly on ticket (no `Dates` struct)
- **No session on ticket**: Sessions are independent — look up via session store/API
- **Tags**: New field on tickets, should be displayed
- **Default paths**: Tickets at `{project}/tickets/`, docs at `{project}/docs/` (not inside `.cortex/`)
- **Comments have `author`** instead of `session_id`
- **Response types**: `TicketResponse` and `TicketSummary` have flat dates, tags, no session field. `TicketSummary` has `HasActiveSession`, `AgentStatus`, `AgentTool` from session enrichment.

## Scope

### 1. `cortex init` command
- Create `tickets/` directory at project root (with `backlog/`, `progress/`, `review/`, `done/` subdirs) instead of `.cortex/tickets/`
- Create `docs/` directory at project root instead of `.cortex/docs/`
- Still create `.cortex/` for config, sessions.json, and other cortex internals
- Respect custom paths from `cortex.yaml` if already configured

### 2. Kanban TUI (`internal/cli/tui/kanban/`)
- Verify ticket cards render correctly with new types
- Display `type` badge on cards (already may exist)
- Display `tags` on cards if space permits
- Due date rendering: use flat `Due` field
- Session status indicators: use `HasActiveSession`, `AgentStatus`, `AgentTool` from summary (these still exist on `TicketSummary`)
- Orphaned session detection still works via `IsOrphaned` on summary

### 3. Ticket Detail TUI (`internal/cli/tui/ticket/`)
- Render flat dates: `Created`, `Updated` (no more `Progress`, `Reviewed`, `Done` timestamps)
- Render `Due` date
- Render `Tags`
- Comment rendering: show `Author` instead of session ID
- Session section: fetch from session API if active, or show "No active session"
- Kill/approve actions: use session API, not ticket-embedded session

### 4. Dashboard TUI (`internal/cli/tui/dashboard/`)
- Verify project listing works with new paths
- Ticket counts should still work (fetched from API)
- Active session indicators from summary data

### 5. SDK Client (`internal/cli/sdk/client.go`)
- Verify all methods work with updated request/response types
- Session-related methods use correct endpoints
- Any new endpoints from ticket 2c (e.g., `GET /sessions/`, `POST /docs/{id}/comments`) should already be in SDK from that ticket

### 6. CLI Commands (`cmd/cortex/commands/`)
- `ticket list` — verify output format with new fields
- `ticket spawn` — verify it works with session store
- Any other commands that reference ticket/doc data

## Goals

- All TUIs render correctly with new data types
- `cortex init` creates correct directory structure at project root
- No references to old `Dates` struct, old session model
- `make build && make lint && make test` pass

## Branch

Working on `feat/frontmatter-storage` branch.