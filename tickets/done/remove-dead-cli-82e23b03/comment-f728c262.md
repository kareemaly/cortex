---
id: f728c262-f7ee-4f9f-9437-d11923b0a612
author: claude
type: review_requested
created: 2026-02-14T13:01:10.304081Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/de84b159-524a-4b2f-865f-e8b6d7ae5e62
        commit: 847e3da
---
## Summary

Removed 8 dead CLI commands and the `internal/upgrade/` package (5 files), totaling -1913 lines.

### Deleted commands
- `cortex show [id]` — duplicated by `cortex ticket <id>` (TUI)
- `cortex ticket list` — duplicated by `cortex kanban` (TUI)
- `cortex ticket spawn <id>` — duplicated by architect's `spawnSession` MCP tool
- `cortex projects` — duplicated by meta's `listProjects` MCP tool
- `cortex register` / `cortex unregister` — duplicated by meta MCP tools
- `cortex config show` — no remaining use case (parent `cortex config` also removed)
- `cortex upgrade` — broken/not needed pre-release

### Deleted package
- `internal/upgrade/` (5 files) — only consumer was the deleted `upgrade` command

### Edits
- **`eject.go`**: Added `formatPath()` helper (moved from deleted `config_show.go`)
- **`ticket.go`**: Removed `ticketListCmd` and `ticketSpawnCmd` registrations, updated description
- **`CLAUDE.md`**: Removed 7 command rows from CLI table, updated debugging reference (`cortex ticket list` → `cortex kanban`), removed `cortex projects` from project registry description
- **`README.md`**: Removed 5 command rows from commands table

### Verification
- `make build` — passes
- `make lint` — 0 issues
- `make test` — all unit tests pass
- `cortex --help` / `cortex ticket --help` — removed commands absent
- No SDK methods became dead (all still used by TUI/MCP/spawn)