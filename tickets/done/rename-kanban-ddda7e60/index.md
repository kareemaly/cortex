---
id: ddda7e60-2d1f-4726-8f46-b88175d6c4ff
title: Rename kanban → project and defaults upgrade → upgrade
type: work
tags:
    - cleanup
    - cli
created: 2026-02-14T12:50:28.787795Z
updated: 2026-02-14T12:59:25.032094Z
---
## Problem

Two commands have names that don't match user intent:

1. **`cortex kanban`** — "kanban" is an implementation detail. Users think "I want to see my project", not "I want a kanban board". Should be `cortex project`.
2. **`cortex defaults upgrade`** — nested under a `defaults` parent that has no other subcommands. The self-update `cortex upgrade` is being removed (separate ticket). This should become simply `cortex upgrade`.

## Requirements

### `cortex kanban` → `cortex project`
- Rename the command from `kanban` to `project`
- Same behavior — launches the kanban/docs/config TUI from the current project folder
- Update any references in help text, documentation, or other commands

### `cortex defaults upgrade` → `cortex upgrade`
- Move `defaults upgrade` to be the top-level `upgrade` command
- Remove the `defaults` parent command
- Keep all existing flags (`--dry-run`, `--yes`)
- Update help text to reflect it's now the primary upgrade command

## Acceptance Criteria

- `cortex project` launches the project TUI (kanban/docs/config tabs)
- `cortex kanban` no longer exists
- `cortex upgrade` refreshes embedded defaults
- `cortex defaults upgrade` no longer exists
- Help text and any cross-references are updated
- Build passes, tests pass