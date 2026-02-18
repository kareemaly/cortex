---
id: 6f4177e0-9d4b-490c-8675-f7b12b45dc59
author: claude
type: done
created: 2026-02-12T11:28:50.397618Z
---
Added a one-line guard in `internal/cli/tui/kanban/column.go:125` to only show OVERDUE and DUE SOON due date badges on backlog tickets. Previously these badges appeared on all columns including done, which was misleading. No documentation changes needed — this is a minor UI behavior fix. Build, lint, and all tests pass. Pushed to main as commit 5ebf962.