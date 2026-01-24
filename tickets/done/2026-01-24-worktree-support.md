# Worktree Support

## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Overview

Enable parallel ticket work by spawning each ticket agent in its own git worktree.

**Depends on:** `ticket-workflow-v2` (prompt templates, concludeSession)

---

## Session Schema

Add to Session struct:
```go
WorktreePath   *string `json:"worktree_path,omitempty"`
FeatureBranch  *string `json:"feature_branch,omitempty"`
```

---

## Spawn Flow (when `git.worktrees: true` in config)

1. Create worktree:
   ```bash
   git worktree add ~/.cortex/worktrees/<session-id> -b ticket/<slug>
   ```

2. Store `worktree_path` and `feature_branch` in session

3. Spawn agent in worktree directory (not project root)

4. Use `ticket-worktree.md` prompt instead of `ticket.md`

---

## Approve Flow (worktree)

When approve triggered for worktree session:
- Use `approve-worktree.md` instead of `approve.md`
- Agent merges to main, pushes, calls `concludeSession`

---

## Cleanup (on concludeSession)

When `concludeSession` called for worktree session:
```bash
git worktree remove <path> --force
git branch -D ticket/<slug>
```

Run from project root (not worktree).

---

## Implementation

### Commits
- `e5c8e4b` feat: implement worktree support for parallel ticket work
- `61f93c3` Merge branch 'ticket/2026-01-24-worktree-support'

### Key Files Changed
| File | Change |
|------|--------|
| `internal/ticket/ticket.go` | Added `WorktreePath`, `FeatureBranch` to Session |
| `internal/ticket/store.go` | Updated `SetSession` signature with worktree params |
| `internal/project/config/config.go` | Added `Worktrees bool` to GitConfig |
| `internal/worktree/worktree.go` | New: Manager with Create/Remove methods |
| `internal/core/spawn/spawn.go` | Added UseWorktree to SpawnRequest, worktree creation flow |
| `internal/daemon/mcp/tools_architect.go` | Pass UseWorktree based on config |
| `internal/daemon/mcp/tools_ticket.go` | Cleanup worktree in concludeSession |

### Decisions
- Worktrees created at `~/.cortex/worktrees/<session-id>` (global, not project-relative)
- Branch naming: `ticket/<slug>` where slug is generated from ticket title
- Worktree-specific prompt templates (`ticket-worktree.md`, `approve-worktree.md`) fall back to standard templates if not present
- Cleanup on `concludeSession` logs errors but doesn't fail the operation

### Scope
Implemented as specified. Prompt template paths already existed in `internal/prompt/prompt.go`.
