---
id: a83fd7bc-1a68-4584-8688-501dab1f28aa
author: claude
type: review_requested
created: 2026-02-02T15:31:56.169304Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Summary

Implemented the reject flow that mirrors the existing approve flow. When rejecting a session review, the REJECT.md instructions are sent to the agent telling it to cleanly rollback all changes and conclude the session.

## Changes Made

### 1. Created REJECT.md Template
- **File:** `internal/install/defaults/claude-code/prompts/ticket/work/REJECT.md`
- Contains rollback instructions for the agent (discard uncommitted changes, revert commits, verify clean state, call concludeSession)

### 2. Added StageReject Constant
- **File:** `internal/prompt/prompt.go`
- Added `StageReject = "REJECT"` constant

### 3. Added Reject API Handler
- **File:** `internal/daemon/api/sessions.go`
- Added `Reject` method that mirrors `Approve`:
  - Finds session and validates it's active
  - Loads and renders REJECT.md template
  - Sends to agent pane via tmux
  - Focuses tmux window

### 4. Registered Reject Route
- **File:** `internal/daemon/api/server.go`
- Added `r.Post("/{id}/reject", sessionHandlers.Reject)`

### 5. Added RejectSession SDK Method
- **File:** `internal/cli/sdk/client.go`
- Added `RejectSession(id string) error` method

### 6. Updated TUI rejectSession
- **File:** `internal/cli/tui/ticket/model.go`
- Changed from adding a comment to calling `m.client.RejectSession(m.ticket.Session.ID)`

## Verification

- `make build` - Passed
- `make lint` - 0 issues
- `make test` - All tests passed