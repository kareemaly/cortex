---
id: 962fad04-899b-48ca-99b9-79b476ec7bc2
author: claude
type: done
created: 2026-02-07T10:13:15.30872Z
---
## Summary

Updated CLI and TUI components for the new frontmatter + directory-per-entity storage layer.

### Changes (9 files, 141 additions, 32 deletions)

**Infrastructure:**
- `internal/install/install.go` — `cortex init` creates `tickets/` and `docs/` at project root, respecting custom paths from cortex.yaml
- `internal/daemon/api/middleware.go` — Validates `.cortex/` directory (canonical project marker) instead of `.cortex/tickets`
- `internal/daemon/api/store_manager.go` — Uses `projectconfig.Load()` + `cfg.TicketsPath()` to resolve tickets path, consistent with DocsStoreManager

**TUI:**
- `internal/cli/tui/kanban/column.go` + `styles.go` — Displays `#tags` on ticket cards in metadata line
- `internal/cli/tui/ticket/model.go` — Tags in attributes panel, comment Author in headers, SESSION section with agent/status/tool from ListSessions() API

**CLI:**
- `cmd/cortex/commands/ticket_list.go` — Added TYPE column to `cortex ticket list`

**Tests:**
- `internal/daemon/mcp/tools_test.go` + `internal/notifications/dispatcher_test.go` — Updated to use new default tickets path (`{root}/tickets/`)

### Verification
- `make build` — passes
- `make lint` — 0 issues
- `make test` — all tests pass

### Commit
`92fbada` on `feat/frontmatter-storage` branch