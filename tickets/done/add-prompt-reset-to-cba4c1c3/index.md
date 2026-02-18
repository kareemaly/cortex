---
id: cba4c1c3-6440-4567-a24c-7eb182b1f473
title: Add prompt reset-to-default in Config tab TUI
type: work
tags:
    - tui
    - configuration
    - api
created: 2026-02-12T14:33:13.803314Z
updated: 2026-02-12T14:47:17.391038Z
---
## Problem

Ejected prompts in the Config tab are currently one-way — once ejected, there's no way to revert to the built-in default without manually deleting the file. Users need a way to reset an ejected prompt back to its default.

## Requirements

### API Layer
- Add a new endpoint to reset/delete an ejected prompt file, so it falls back to the built-in default
- Add corresponding SDK client method

### TUI Layer
- Add `x` shortcut in the Config tab to reset a prompt to default
- `x` should only be active on **ejected** prompts (● ejected badge). On default prompts, show a status message like "already using default"
- Show a confirmation modal before resetting: `Reset "group/filename.md" to default?  [y]es  [n]o`
- Follow the existing confirmation modal pattern used in the kanban (orphaned session deletion uses a two-step modal with y/n)
- After successful reset, refresh the prompt list so the badge updates back to `○ default`
- Update the help bar to include the new shortcut

## Acceptance Criteria
- Pressing `x` on an ejected prompt shows a y/n confirmation dialog
- Confirming deletes the ejected file and the prompt reverts to the built-in default
- The prompt list refreshes and shows `○ default` badge after reset
- Pressing `x` on an already-default prompt shows a brief status message (no action taken)
- Help text is updated to reflect the new shortcut
- The new API endpoint is accessible via the SDK client