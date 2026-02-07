---
id: 9e746cf5-f814-4c20-9f14-7c02289e5c6e
author: claude
type: ticket_done
created: 2026-01-27T10:04:19.352302Z
---
## Summary

Added automatic worktree and branch cleanup to the `Fresh()` spawn mode, fixing the issue where re-spawning a fresh session would fail with `fatal: a branch named 'ticket/xxx' already exists`.

## Problem

When `Fresh()` in `spawn.go` ended an old session via `EndSession()`, it only marked the session as ended in the store — it did not remove the git worktree or delete the branch. When `Spawn()` then ran `git worktree add -b ticket/{slug}`, it failed because the branch already existed from the previous session.

## Changes Made

**File modified:** `internal/core/spawn/spawn.go` (9 lines added)

In the `Fresh()` method (line 399-406), after `EndSession()` marks the old session as ended, the code now:
1. Reads the old session's `WorktreePath` and `FeatureBranch` pointer fields
2. If both are non-nil, creates a `worktree.Manager` and calls `Remove()` which executes `git worktree remove --force` and `git branch -D`
3. Logs a warning on cleanup failure rather than returning an error, since the worktree/branch may already be partially cleaned up

## Key Decisions

- **Warning on failure, not error**: Cleanup failure is logged as a warning rather than blocking the spawn. The worktree or branch may already be partially removed, and `Spawn()` should still attempt to proceed.
- **Nil-guarded**: Only attempts cleanup if both `WorktreePath` and `FeatureBranch` are set on the old session (both are `*string`). Sessions without worktrees (e.g., architect sessions) are unaffected.
- **Used existing `worktree.Manager.Remove()`**: No new git logic needed — reused the existing method that already handles force-removal and branch deletion.

## Verification

- `make build` — compiles successfully
- `make test` — all unit tests pass
- `make lint` — 0 issues

## Follow-up

None required. Normal and resume spawn modes are unaffected by this change.