# Approve Worktree Fix and Review Status

## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Problem 1: Approve uses wrong prompt for worktree sessions

`sessions.go:151` always loads `approve.md`:
```go
approvePath := prompt.ApprovePath(projectPath)  // always non-worktree
```

Should check `session.WorktreePath` and use `ApproveWorktreePath()` for worktree sessions.

## Problem 2: requestReview doesn't move ticket to review

`handleRequestReview` in `tools_ticket.go` only adds to `Session.RequestedReviews` but doesn't move the ticket to review status.

## Requirements

1. In approve handler (`sessions.go`):
   - Check if session has `WorktreePath`
   - If yes, use `prompt.ApproveWorktreePath()`
   - If no, use `prompt.ApprovePath()`

2. In `handleRequestReview` (`tools_ticket.go`):
   - After adding review request, move ticket to review status
   - Only move if not already in review or done

## Implementation

### Commits

- `08da38d` fix: approve handler uses worktree prompt and requestReview moves ticket to review

### Key Files Changed

- `internal/daemon/api/sessions.go` - Added conditional to use `approve-worktree.md` when session has `WorktreePath`
- `internal/daemon/mcp/tools_ticket.go` - Added move to review status after `AddReviewRequest` call
- `internal/daemon/api/integration_test.go` - Removed stale `HookExecutor` reference (bonus fix)

### Decisions

- Used idempotent check before moving: skip if already in review or done status
- Pattern matches existing worktree handling in `spawn.go:427-433`
