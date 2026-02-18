---
id: 82e23b03-04e6-4379-aea1-46b4d86b8977
title: Remove dead CLI commands for public release
type: work
tags:
    - cleanup
    - cli
created: 2026-02-14T12:50:22.005954Z
updated: 2026-02-14T13:04:12.867159Z
---
## Problem

Several CLI commands are redundant — covered by TUIs or the architect — and need to be removed before public release to keep the surface clean.

## Commands to Remove

1. **`cortex show [id]`** — duplicate of `cortex ticket <id>`, same implementation
2. **`cortex ticket list`** — architect handles listing, kanban TUI shows board
3. **`cortex ticket spawn <id>`** — architect handles spawning via MCP tools
4. **`cortex projects [--json]`** — covered by `cortex dashboard` TUI
5. **`cortex register [path]`** — covered by dashboard TUI
6. **`cortex unregister [path]`** — covered by dashboard TUI
7. **`cortex config show`** — covered by config tab in project TUI
8. **`cortex upgrade` (self-update)** — broken, never tested, removing entirely (the name will be repurposed for defaults upgrade in a separate ticket)

## Requirements

- Remove all command registrations and their implementation files
- Remove any SDK client methods that become unused after removal (e.g., if `ListAllTickets` was only used by `ticket list`)
- Keep `cortex ticket <id>` working — only remove the `list` and `spawn` subcommands
- Ensure `cortex ticket` without arguments shows help or errors cleanly

## Acceptance Criteria

- None of the listed commands are available in the CLI
- `cortex --help` shows a clean command list without the removed commands
- `cortex ticket <id>` still works
- Build passes, tests pass