---
id: fcb37d2b-06b1-4cba-8606-f34324a3489f
title: Add Config tab to architect companion TUI
type: work
tags:
    - tui
    - configuration
created: 2026-02-12T11:50:15.138153Z
updated: 2026-02-12T12:11:34.389479Z
---
## Overview

Add a new "Config" tab to the architect companion TUI (currently `cortex kanban` with Kanban and Docs tabs). This tab lets users browse, eject, and edit project prompts and configuration.

## Layout

Three vertical sections:

1. **cortex.yaml** — navigable item at the top, editable via `$EDITOR`
2. **Prompt list** — all prompt files grouped by section headers, with ejected/default status
3. **Preview pane** — always visible at the bottom, shows content of the currently highlighted file

### Section headers and grouping

Prompts are grouped under styled, non-navigable section headers that match the filesystem hierarchy:

- `ARCHITECT` — architect/KICKOFF.md, architect/SYSTEM.md
- `META` — meta/KICKOFF.md, meta/SYSTEM.md
- `TICKET › WORK` — ticket/work/APPROVE.md, KICKOFF.md, SYSTEM.md
- `TICKET › DEBUG` — ticket/debug/APPROVE.md, KICKOFF.md, SYSTEM.md
- `TICKET › RESEARCH` — ticket/research/APPROVE.md, KICKOFF.md, SYSTEM.md
- `TICKET › CHORE` — ticket/chore/APPROVE.md, KICKOFF.md, SYSTEM.md

The actual groups should be derived from whatever exists in the agent's defaults directory (e.g. `~/.cortex/defaults/claude-code/prompts/` or `~/.cortex/defaults/opencode/prompts/`), not hardcoded.

### Status indicators

Each file shows its ejection status:
- `○ default` — using the default from `~/.cortex/defaults/{agent}/prompts/`
- `● ejected` — customized copy exists in `.cortex/prompts/`

## Navigation

- `j/k` or `↑/↓` — navigate between files only (section headers are skipped)
- `cortex.yaml` at the top is also a navigable item
- Preview pane updates as the cursor moves

## Actions

- **`e` on a default prompt** → ejects it (copies to `.cortex/prompts/`, status changes to ejected)
- **`e` on an ejected prompt** → opens `$EDITOR` in a tmux popup to edit
- **`e` on cortex.yaml** → opens `$EDITOR` in a tmux popup to edit
- **`c`** — shortcut to edit `cortex.yaml` (same as navigating to it and pressing `e`)

## Integration

- This is a new tab alongside the existing Kanban and Docs tabs in the companion TUI
- Accessible via the same tab switching mechanism (`tab`/`[`/`]`)
- The prompt list should read from the project config to determine which agent type is active, then scan both the defaults directory and the project's `.cortex/prompts/` to determine ejection status

## Acceptance Criteria

- Config tab appears in the companion TUI tab bar
- All prompt files from the active agent's defaults are listed with correct grouping
- Ejection status is correctly displayed for each file
- `e` ejects default prompts and edits ejected prompts via tmux popup
- `c` opens cortex.yaml for editing via tmux popup
- Preview pane shows file content and updates on navigation
- Tab switching between Kanban, Docs, and Config works