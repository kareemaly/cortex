---
id: d759719d-2c3b-4b63-b7ef-e7da1341d4e3
author: claude
type: done
created: 2026-02-05T07:08:06.240148Z
---
## Summary

Implemented multiple ticket types (debug, research, chore) with specialized workflows.

### New Ticket Types

| Type | Purpose | Workflow |
|------|---------|----------|
| `debug` | Root cause analysis | Reproduce → Investigate → Document → Fix |
| `research` | Exploration | Read-only, brainstorm, document findings |
| `chore` | Maintenance | Minimal ceremony, direct execution |

### Files Created (9)

- `internal/install/defaults/claude-code/prompts/ticket/debug/{SYSTEM,KICKOFF,APPROVE}.md`
- `internal/install/defaults/claude-code/prompts/ticket/research/{SYSTEM,KICKOFF,APPROVE}.md`
- `internal/install/defaults/claude-code/prompts/ticket/chore/{SYSTEM,KICKOFF,APPROVE}.md`

### Files Modified (7)

- `internal/install/defaults/claude-code/cortex.yaml` - Added type configs with appropriate permissions
- `internal/cli/tui/kanban/styles.go` - Added type badge color styles
- `internal/cli/tui/kanban/column.go` - Added type badge rendering before titles
- `internal/daemon/api/tickets.go` - Added type validation against project config
- `internal/install/embed_test.go` - Added 9 new prompt files to expected list
- `internal/daemon/mcp/types.go` - Updated CreateTicketInput type description
- `README.md` - Updated config example to show available ticket types

### Verification

- `make build` ✓
- `make lint` ✓ (0 issues)
- `make test` ✓ (all tests pass)

### Commit

`3c71f58` - feat: add multiple ticket types (debug, research, chore)