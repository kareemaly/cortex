---
id: 342259ea-58fd-4c37-9502-903a8f6519c1
author: claude
type: review_requested
created: 2026-02-14T12:57:57.273121Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/a00f2d6a-c662-40af-887c-6b82fbab609f
        commit: ceda99e
---
## Summary

Renamed two CLI commands to better match user intent:

1. **`cortex kanban` → `cortex project`**: The kanban name described an implementation detail. Renamed the command file (`kanban.go` → `project.go`), variable (`kanbanCmd` → `projectCmd`), and updated the two companion pane references in `internal/core/spawn/spawn.go`. Internal TUI packages (`internal/cli/tui/kanban/`) left unchanged as they correctly describe the board component.

2. **`cortex upgrade`**: Replaced the self-update binary logic with defaults upgrade logic (previously at `cortex defaults upgrade`). Deleted the parent `defaults.go` command and `defaults_upgrade.go`, consolidating all logic into `upgrade.go` with `--dry-run` and `--yes` flags.

## Files changed

| File | Action |
|------|--------|
| `cmd/cortex/commands/kanban.go` → `project.go` | Renamed, updated command name/short |
| `cmd/cortex/commands/upgrade.go` | Replaced self-update with defaults upgrade logic |
| `cmd/cortex/commands/defaults.go` | Deleted |
| `cmd/cortex/commands/defaults_upgrade.go` | Deleted (moved to upgrade.go) |
| `internal/core/spawn/spawn.go` | 2 companion pane refs updated |
| `CLAUDE.md` | Updated command table |
| `README.md` | Updated command table |

## Verification

- `make build` — compiles successfully
- `make lint` — 0 issues
- `make test` — all unit tests pass
- `cortex project --help` — correct help text
- `cortex upgrade --help` — shows defaults upgrade with --dry-run/--yes
- `cortex kanban` — "unknown command" (removed)
- `cortex defaults upgrade` — "unknown command" (removed)