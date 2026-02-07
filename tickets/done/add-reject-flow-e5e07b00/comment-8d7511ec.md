---
id: 8d7511ec-3f11-4aba-a47c-eb29e20577ca
author: claude
type: done
created: 2026-02-02T15:41:44.996232Z
---
## Summary

Implemented the reject flow for session reviews, mirroring the existing approve flow.

## Changes Made

1. **Created REJECT.md template** (`internal/install/defaults/claude-code/prompts/ticket/work/REJECT.md`)
   - Contains rollback instructions: discard uncommitted changes, revert commits, verify clean state, call concludeSession

2. **Added StageReject constant** (`internal/prompt/prompt.go`)
   - Added `StageReject = "REJECT"` to stage constants

3. **Added Reject API handler** (`internal/daemon/api/sessions.go`)
   - Finds session and validates it's active
   - Loads and renders REJECT.md template with fallback
   - Sends to agent pane via tmux
   - Focuses tmux window

4. **Registered reject route** (`internal/daemon/api/server.go`)
   - Added `POST /sessions/{id}/reject` endpoint

5. **Added RejectSession SDK method** (`internal/cli/sdk/client.go`)
   - New `RejectSession(id string) error` method

6. **Updated TUI rejectSession** (`internal/cli/tui/ticket/model.go`)
   - Changed from adding a comment to calling SDK's RejectSession

## Verification

- Build: Passed
- Lint: 0 issues
- Tests: All passed

## Commit

`06dd273` feat(api): add reject flow for session reviews