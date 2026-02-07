---
id: 0a43bb2a-594a-497d-a84e-dfc990c70c04
author: claude
type: done
created: 2026-02-05T11:39:56.505907Z
---
## Summary

Added diff preview and confirmation prompt to `cortex defaults upgrade` to prevent unexpected overwrites of user customizations.

## Changes Made

### `internal/install/embed.go`
- Added `EmbeddedContent` and `DiskContent` fields to `CompareItem` struct
- Updated `compareEmbeddedDir` to populate content fields for files that will be updated or created

### `cmd/cortex/commands/defaults_upgrade.go`
- Added `--yes`/`-y` flag to skip confirmation prompt (for automation)
- Rewrote upgrade flow to:
  1. Collect all changes across configs
  2. Display file list with status indicators (✓ unchanged, • will update, + will create)
  3. Show unified diff for files being updated
  4. Print summary
  5. Prompt for confirmation (unless `--yes`)
  6. Apply changes only if confirmed
- Added non-TTY detection - requires `--yes` when stdin is piped
- Implemented `generateUnifiedDiff` helper with 3-line context

## New Behavior
- `cortex defaults upgrade` - shows diff preview, prompts `[y/N]`
- `cortex defaults upgrade --yes` - applies without prompt
- `cortex defaults upgrade --dry-run` - unchanged
- Piped input without `--yes` → error message

## Testing
- Build passes
- Lint passes (0 issues)
- Unit tests pass
- Manual testing verified all scenarios

Merged to main and pushed.