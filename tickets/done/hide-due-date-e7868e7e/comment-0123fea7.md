---
id: 0123fea7-0fbf-488e-9778-9c259d451519
author: claude
type: review_requested
created: 2026-02-12T11:28:30.217179Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
        commit: 5ebf962
---
One-line change in `internal/cli/tui/kanban/column.go` (line 125): added `&& c.status == "backlog"` guard to the due date badge condition, so OVERDUE and DUE SOON badges only render in the backlog column. Build and all unit tests pass.