---
id: e646eed1-8aa1-47e9-8fc9-e595f1fae004
title: 'Fix ticket popup: update removed "cortex show" to "cortex ticket"'
type: debug
tags:
    - debug
    - tui
    - cleanup
created: 2026-02-15T08:39:08.77203Z
updated: 2026-02-15T08:51:28.915523Z
---
## Problem

The tmux ticket detail popup launched from the project TUI (kanban) is broken. When pressing `o` or `enter` on a ticket, nothing happens because the popup invokes a CLI command that was removed in the recent cleanup (commit `847e3da`).

## Root Cause

In `internal/daemon/api/tickets.go`, the `ShowPopup` handler constructs the command:
```go
command := fmt.Sprintf("cortex show %s", ticketID)
```

The `cortex show` command was deleted as part of the CLI cleanup. The replacement is `cortex ticket <id>`.

## Fix

Change the command string from `"cortex show %s"` to `"cortex ticket %s"` in the ShowPopup handler.

## Acceptance Criteria
- Pressing `o` or `enter` on a ticket in the project TUI opens the ticket detail popup
- The popup shows the ticket detail view correctly