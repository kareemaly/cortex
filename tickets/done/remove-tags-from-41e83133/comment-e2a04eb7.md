---
id: e2a04eb7-7fbe-4a79-ae9c-bd7b5bb13129
author: claude
type: done
created: 2026-02-12T09:51:40.573205Z
---
## Summary

Removed tags from kanban TUI ticket cards.

### Changes
- **`internal/cli/tui/kanban/column.go`** — Removed `tagsStr` construction block and both conditionals that appended tags to the metadata line (selected and normal ticket views). Updated comments accordingly.
- **`internal/cli/tui/kanban/styles.go`** — Removed unused `tagsStyle` declaration.

### Result
Ticket cards in the kanban board now display only `[agent-status · ]creation-date` in their metadata line, with no tag display.

### Verification
- `make build` — compiles cleanly
- `make lint` — 0 issues
- `make test` — all tests pass
- Pushed to origin/main (commit 99d2a11)