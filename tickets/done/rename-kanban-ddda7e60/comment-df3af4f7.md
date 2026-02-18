---
id: df3af4f7-0169-44a5-9820-a4cf1f1254af
author: claude
type: done
created: 2026-02-14T12:59:25.031208Z
---
## Summary

Renamed two CLI commands to better match user intent:

### Changes

1. **`cortex kanban` → `cortex project`**
   - Renamed `cmd/cortex/commands/kanban.go` → `project.go`
   - Updated variable `kanbanCmd` → `projectCmd`, Use/Short text
   - Updated 2 companion pane references in `internal/core/spawn/spawn.go` (lines 551, 1016)
   - Internal TUI packages (`internal/cli/tui/kanban/`) left unchanged — they correctly describe the board component

2. **`cortex upgrade` (was self-update) → defaults upgrade**
   - Replaced self-update binary logic in `upgrade.go` with defaults upgrade logic
   - Deleted `cmd/cortex/commands/defaults.go` (parent command no longer needed)
   - Deleted `cmd/cortex/commands/defaults_upgrade.go` (logic moved into `upgrade.go`)
   - New flags: `--dry-run` / `--yes` (replacing old `--check` / `--version`)

3. **Documentation**
   - Updated `CLAUDE.md` command table: renamed kanban→project, updated upgrade description, removed defaults upgrade row
   - Updated `README.md` command table: renamed kanban→project, updated upgrade description

### Verification

- `make build` — compiles
- `make lint` — 0 issues
- `make test` — all pass
- `cortex project --help` / `cortex upgrade --help` — correct output
- `cortex kanban` / `cortex defaults upgrade` — "unknown command" (removed)
- Pre-push hooks passed, merged to main and pushed