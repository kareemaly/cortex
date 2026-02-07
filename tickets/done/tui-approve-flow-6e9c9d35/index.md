---
id: 6e9c9d35-b4aa-4e1f-b1dc-13a292bd112c
title: TUI Approve Flow
type: ""
created: 2026-01-24T14:41:51Z
updated: 2026-01-24T14:41:51Z
---
## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Overview

Extend approve flow to ticket detail view and display review requests.

**Note:** Kanban 'a' key approve action was implemented in `ticket-workflow-v2`.

---

## Show Review Requests

In ticket detail view, display `Session.RequestedReviews`:
```
─── Review Requests ───
[repo: .]  "Implemented feature X, added tests"  (2 min ago)
[repo: ../lib]  "Updated shared types"  (1 min ago)
```

---

## Ticket Detail Approve Action

Add 'a' key to ticket detail view (same behavior as kanban):
1. Call POST /sessions/{id}/approve API
2. Focus agent's tmux window

Note: Worktree detection (approve-worktree.md vs approve.md) will work once worktree-support ticket adds Session.WorktreePath field.

---

## Implementation

### Commits Pushed
- `51d902c` feat: add approve flow with review requests display to ticket detail TUI

### Key Files Changed
- `internal/daemon/api/types.go` - Added `RequestedReviewResponse` struct, updated `SessionResponse` and `toSessionResponse()`
- `internal/daemon/api/sessions.go` - Added `FocusWindow()` call after approve prompt
- `internal/cli/sdk/client.go` - Added `RequestedReviewResponse` struct and field to `SessionResponse`
- `internal/cli/tui/ticket/keys.go` - Added `KeyApprove` constant, updated `helpText()` with conditional approve display
- `internal/cli/tui/ticket/model.go` - Added approve handling, `renderReviewRequests()`, `formatTimeAgo()` helper

### Decisions Made
- Approve action only shows in help bar when session has review requests (cleaner UX)
- FocusWindow failure is non-fatal (logs warning, continues with success response)
- Review requests display format: `[repo: .]  "Summary"  (2 min ago)`

### Scope Changes
- None - implemented as specified