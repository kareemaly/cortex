---
id: d5ad7c6b-90b1-4072-a804-0fe86483d7a0
author: claude
type: review_requested
created: 2026-02-05T11:39:03.1992Z
action:
    type: git_diff
    args:
        commit: cb67f37
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/c8c63718-fdf8-4ccc-8086-6ca3e884a99f
---
## Summary

Added diff preview and confirmation prompt to `cortex defaults upgrade` command to prevent unexpected overwrites of user customizations.

## Changes

### `internal/install/embed.go`
- Added `EmbeddedContent` and `DiskContent` fields to `CompareItem` struct
- Updated `compareEmbeddedDir` to populate content fields for `CompareWillUpdate` and `CompareWillCreate` items

### `cmd/cortex/commands/defaults_upgrade.go`
- Added `--yes`/`-y` flag to skip confirmation prompt (for automation/scripting)
- Rewrote `runDefaultsUpgradeApplyAll` to:
  1. Collect all changes across configs first
  2. Display file list with status indicators
  3. Show unified diff for files that will be updated
  4. Print summary of changes
  5. Prompt for confirmation (unless `--yes`)
  6. Apply changes only if confirmed
- Added `promptConfirmation` helper for y/n prompts
- Added `generateUnifiedDiff` helper for unified diff generation with context lines
- Added non-TTY detection - requires `--yes` when stdin is piped
- Removed unused `runDefaultsUpgradeApply` function

## Behavior

1. `cortex defaults upgrade` - shows diff preview, prompts for confirmation
2. `cortex defaults upgrade --yes` - applies immediately without prompt
3. `cortex defaults upgrade --dry-run` - unchanged (preview only)
4. `echo "y" | cortex defaults upgrade` - errors asking for `--yes` flag

## Testing
- Build passes (`make build`)
- Lint passes (`make lint`)
- Unit tests pass (`make test`)
- Manual testing verified diff preview, confirmation prompt, `--yes` flag, and non-TTY detection