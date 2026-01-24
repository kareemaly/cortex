# Worktree Context Support

## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Problem

Worktree operations in `internal/worktree/worktree.go` execute Git commands but don't support context cancellation. If an operation takes a long time, the caller cannot cancel it.

Affected methods:
- `Manager.Create()`
- `Manager.Remove()`

## Requirements

- Add `ctx context.Context` parameter to `Create()` and `Remove()` methods
- Pass context to underlying Git command execution
- Update all callers to pass context
