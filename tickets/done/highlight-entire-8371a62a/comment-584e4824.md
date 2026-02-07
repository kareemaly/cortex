---
id: 584e4824-c8bd-4fa7-b074-a9768d8c8cf7
author: claude
type: review_requested
created: 2026-02-02T16:44:39.475079Z
action:
    type: git_diff
    args:
        commit: 8278e7b
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/3d95fb2a-6fb5-4ab2-827f-a6c631a18ee3
---
## Summary
Fixed the comment row selection highlighting in the ticket TUI so the entire row has a background highlight when selected.

## Change
Modified `applyBackgroundToBlock()` in `internal/cli/tui/ticket/model.go`:
- Removed `.Width(width)` from the lipgloss style
- The function already manually pads each line to full width before applying the background
- Using both `.Width()` and manual padding caused rendering issues with pre-styled content containing ANSI escape codes

## Verification
- `make build` passes
- `make lint` passes with 0 issues

## Testing Steps
1. Start the daemon: `cortexd start`
2. Open ticket TUI: `cortex show <ticket-id>`
3. Press Tab/[ to focus the comment list
4. Navigate with j/k to select different comments
5. Verify entire comment row (header + preview lines) has background highlight