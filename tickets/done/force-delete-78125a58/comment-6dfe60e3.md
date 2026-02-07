---
id: 6dfe60e3-04e5-45e1-9107-4216641fc1ff
author: claude
type: progress
created: 2026-01-27T10:03:40.908787Z
---
Implementation complete. Added worktree/branch cleanup to `Fresh()` in `internal/core/spawn/spawn.go`. After `EndSession()`, the code now calls `worktree.Manager.Remove()` to force-remove the old worktree and delete the branch before `Spawn()` recreates them. Build, tests, and lint all pass.