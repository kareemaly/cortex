---
id: 78125a58-6604-4494-bf9d-2031db898590
title: Force Delete Worktree and Branch on Fresh Spawn
type: ""
created: 2026-01-27T09:55:27.465512Z
updated: 2026-01-27T10:04:19.353406Z
---
## Problem

When spawning a session with `mode=fresh`, if a worktree and branch already exist from a previous session, the spawn fails with:

```
git worktree add: fatal: a branch named 'ticket/xxx' already exists
```

The user has to manually remove the worktree (`git worktree remove --force`) and delete the branch (`git branch -D`) before retrying.

## Solution

When `mode=fresh`, the spawn orchestration should automatically clean up any existing worktree and branch for the ticket before creating new ones:

1. Check if a worktree already exists for the ticket
2. If so, force-remove it (`git worktree remove --force`)
3. Delete the existing branch (`git branch -D`)
4. Proceed with normal worktree/branch creation

## Scope

- `internal/core/spawn/orchestrate.go` — Add cleanup logic in the fresh mode path before worktree creation

## Acceptance Criteria

- [ ] `mode=fresh` automatically removes existing worktree and branch for the ticket
- [ ] No manual cleanup required when re-spawning fresh
- [ ] Normal and resume modes are unaffected