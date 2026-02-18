---
id: 30417e9c-8fac-4abc-8d7f-5a169eec3ee9
author: claude
type: done
created: 2026-02-14T13:04:12.866163Z
---
## Summary

Removed dead CLI commands and the `internal/upgrade/` package to clean the command surface area before public release.

### Deleted (in worktree)
- **8 command files**: `show.go`, `ticket_list.go`, `ticket_spawn.go`, `projects.go`, `register.go`, `config_show.go`, `config.go`, `upgrade.go` (old self-update version)
- **5 package files**: `internal/upgrade/` (upgrade.go, binary.go, checksum.go, github.go, upgrade_test.go)
- Net: -1913 lines

### Edited
- **`eject.go`**: Added `formatPath()` helper moved from deleted `config_show.go`
- **`ticket.go`**: Removed `ticketListCmd`/`ticketSpawnCmd`, updated description
- **`CLAUDE.md`**: Removed dead command rows, updated debugging reference, updated project registry text
- **`README.md`**: Removed dead command rows

### Merge Resolution
HEAD had diverged — `cortex kanban` was renamed to `cortex project` and `upgrade.go` was repurposed for defaults refresh. Resolved 3 conflicts:
1. **CLAUDE.md**: Used `cortex project` naming from HEAD, kept dead command removals from branch, kept `cortex upgrade` row with new description
2. **README.md**: Same approach
3. **upgrade.go**: Kept HEAD's repurposed version (defaults refresh, not self-update)

### Verification
- `make build` — passes on both worktree and main
- `make lint` — 0 issues
- `make test` — all unit tests pass
- Pre-push hooks passed (lint + build)