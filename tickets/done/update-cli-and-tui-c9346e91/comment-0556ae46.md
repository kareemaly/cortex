---
id: 0556ae46-3625-4095-9b78-9eeef1a55995
author: claude
type: review_requested
created: 2026-02-07T10:12:41.103125Z
action:
    type: git_diff
    args:
        commit: 92fbada
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Summary

Updated CLI and TUI components for the new frontmatter + directory-per-entity storage layer.

### Changes (9 files, 141 additions, 32 deletions):

**Infrastructure (3 files):**
1. **`internal/install/install.go`** — `cortex init` now creates `tickets/` and `docs/` at project root (not inside `.cortex/`). Writes config first, then loads it to resolve paths (respects custom `tickets.path`/`docs.path` from cortex.yaml).

2. **`internal/daemon/api/middleware.go`** — Project validation now checks for `.cortex/` directory (the canonical project marker) instead of `.cortex/tickets` which is no longer the default location.

3. **`internal/daemon/api/store_manager.go`** — Uses `projectconfig.Load()` + `cfg.TicketsPath()` to resolve tickets directory, consistent with how `DocsStoreManager` already works.

**TUI (3 files):**
4. **`internal/cli/tui/kanban/column.go` + `styles.go`** — Ticket cards now display `#tags` on the metadata line between agent status and date.

5. **`internal/cli/tui/ticket/model.go`** — Three additions:
   - Tags shown in attributes panel after Due date
   - Comment Author displayed in comment header lines
   - SESSION section in attributes panel showing agent, status, and tool (fetched from ListSessions() API)

**CLI (1 file):**
6. **`cmd/cortex/commands/ticket_list.go`** — Added TYPE column to `cortex ticket list` table output.

**Tests (2 files):**
7. **`internal/daemon/mcp/tools_test.go`** + **`internal/notifications/dispatcher_test.go`** — Updated test setup to create ticket stores at `{root}/tickets/` (the new default) and ensure `.cortex/` directory exists for middleware validation.

### Verification
- `make build` — passes
- `make lint` — 0 issues
- `make test` — all tests pass