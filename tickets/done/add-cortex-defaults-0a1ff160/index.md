---
id: 0a1ff160-926f-4ec5-b658-344ceedd0b97
title: Add `cortex defaults upgrade` command
type: work
created: 2026-02-05T09:09:50.261067Z
updated: 2026-02-05T09:31:31.813137Z
---
## Summary

Add a new CLI command to upgrade `~/.cortex/defaults/` with the latest embedded defaults from the binary.

## Context

- Embedded defaults ship in the binary at `internal/install/defaults/claude-code/`
- On first `cortex init`, these are copied to `~/.cortex/defaults/claude-code/`
- When binaries upgrade with new/improved defaults, users have stale copies
- Need a way to refresh defaults without losing project-level customizations (those live in `.cortex/prompts/`)

## Requirements

1. New command: `cortex defaults upgrade`
2. Replaces `~/.cortex/defaults/` contents with current embedded defaults
3. Reuse existing `copyEmbeddedDefaults(..., force=true)` from `internal/install/embed.go`
4. Output should list what files were updated
5. Add `--dry-run` flag to preview changes without applying

## Implementation Notes

- Add new command file: `cmd/cortex/commands/defaults.go` (or `defaults_upgrade.go`)
- Follow existing command patterns (see `eject.go`, `upgrade.go`)
- The `copyEmbeddedDefaults` function already supports `force` parameter
- Consider adding a `defaults` subcommand group for future extensibility (`defaults list`, `defaults diff`)

## Out of Scope

- Project-level defaults (handled by `eject`)
- Merging user edits with new defaults (full replacement is intended)