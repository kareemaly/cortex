---
id: 355c5ac5-129b-4009-b8fa-1c396768768c
author: claude
type: done
created: 2026-02-05T06:40:02.921765Z
---
## Summary

Successfully removed the reject workflow from Cortex to simplify the approval flow.

## Changes Made

**Deleted:**
- `internal/install/defaults/claude-code/prompts/ticket/work/REJECT.md`

**Modified (9 files, -180 lines):**
- `internal/prompt/prompt.go` - Removed `StageReject` constant
- `internal/daemon/api/sessions.go` - Removed `Reject()` handler
- `internal/daemon/api/server.go` - Removed `/sessions/{id}/reject` route
- `internal/cli/sdk/client.go` - Removed `RejectSession()` method
- `internal/cli/tui/ticket/model.go` - Removed reject state, message types, handlers, and method
- `internal/cli/tui/ticket/keys.go` - Removed `[x] reject` from modal help text
- `README.md` - Updated "approves or rejects" → "approves"
- `CONFIG_DOCS.md` - Removed REJECT.md from prompt tree

## Verification
- Build: ✓
- Lint: 0 issues
- Tests: All pass
- No reject references remain in codebase (verified via grep)

## Commit
`b2f967a` - refactor: remove reject workflow from Cortex

Pushed to origin/main.