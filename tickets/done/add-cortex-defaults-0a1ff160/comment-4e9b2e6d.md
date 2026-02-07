---
id: 4e9b2e6d-aa4d-4076-a9be-ad8eb3934cd0
author: claude
type: done
created: 2026-02-05T09:31:31.811785Z
---
## Summary

Added `cortex defaults upgrade` command to refresh `~/.cortex/defaults/` with the latest embedded defaults from the binary.

## Changes

### New Files
- `cmd/cortex/commands/defaults.go` - Parent command group
- `cmd/cortex/commands/defaults_upgrade.go` - Upgrade subcommand with `--dry-run` flag

### Modified Files
- `internal/install/embed.go` - Exported `CopyEmbeddedDefaults`, added `CompareEmbeddedDefaults` for dry-run comparison
- `internal/install/install.go` - Updated call to use exported function
- `internal/install/embed_test.go` - Updated test function calls
- `CLAUDE.md` - Added command to CLI Commands table

## Usage
- `cortex defaults upgrade` - Updates all default files (force overwrites)
- `cortex defaults upgrade --dry-run` - Preview changes without applying

## Commits
- `13a7c90` feat: add cortex defaults upgrade command
- `91fa453` docs: add cortex defaults upgrade to CLI commands table