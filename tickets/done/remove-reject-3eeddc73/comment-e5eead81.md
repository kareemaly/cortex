---
id: e5eead81-35d1-4fcc-ad49-fd94ba44d94f
author: claude
type: review_requested
created: 2026-02-05T06:39:29.279647Z
action:
    type: git_diff
    args:
        commit: b2f967a
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Summary

Removed the reject workflow entirely from Cortex to simplify the approval flow. The reject feature was a variation of approve that sent different instructions to the agent (discard changes vs ship changes). Both paths led to the same `done` state via `concludeSession`, so removing reject simplifies the workflow.

## Changes Made

### Files Deleted
- `internal/install/defaults/claude-code/prompts/ticket/work/REJECT.md` - Reject prompt template

### API Layer
- `internal/daemon/api/sessions.go` - Removed `Reject()` handler (~105 lines)
- `internal/daemon/api/server.go` - Removed `/sessions/{id}/reject` route

### SDK Client
- `internal/cli/sdk/client.go` - Removed `RejectSession()` method (~20 lines)

### Prompt Constants
- `internal/prompt/prompt.go` - Removed `StageReject` constant

### TUI Components
- `internal/cli/tui/ticket/model.go` - Removed:
  - `SessionRejectedMsg` and `RejectErrorMsg` structs
  - `rejecting` field from Model
  - Message handlers for reject messages
  - `rejectSession()` method
  - Reject loading state render
  - 'x' key handler for reject in review modal
- `internal/cli/tui/ticket/keys.go` - Removed `[x] reject` from modal help text

### Documentation
- `README.md` - Changed "approves or rejects" to "approves"
- `internal/install/defaults/claude-code/CONFIG_DOCS.md` - Removed REJECT.md from prompt tree

## Verification

```bash
# No reject references remain
grep -ri "reject" --include="*.go" --include="*.md" . | grep -v ".git"
# Returns no matches

# Build passes
make build  # ✓

# Lint passes
make lint   # 0 issues

# Tests pass
make test   # All tests pass
```

## Notes
- Approval flow still works: approve → concludeSession → done
- No lifecycle hooks were tied to reject (only pickup, review, approve)
- Historical ticket JSON files are unaffected (just history)