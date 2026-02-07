---
id: b03b8fac-f983-48ea-b641-e4b6695d75ca
author: claude
type: review_requested
created: 2026-02-05T11:43:01.99848Z
action:
    type: git_diff
    args:
        commit: "24733e6"
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/b29838ce-8b11-43cd-959c-7060ad636178
---
## Summary

Verified that architect sessions already correctly use the 30%/70% split pane layout. The implementation in `internal/tmux/pane.go:SplitWindowHorizontal()` uses `tmux split-window -h -p 70` which gives 70% to the new (right) pane and 30% to the original (left) pane.

**Key findings:**
- The code path `cortex architect` → `POST /architect/spawn` → `spawner.Spawn()` → `SpawnArchitect()` → `SplitWindowHorizontal()` correctly passes the companion command ("cortex kanban") which triggers the split
- Manual tmux test confirmed `-p 70` produces the expected 30%/70% ratio
- If users see 50%/50% split, it's likely from an old session before this code was in place

**Changes made:**
Added pane width verification to integration tests to catch any future regressions:

1. `TestIntegrationSplitWindowHorizontal` - now verifies ~30%/70% split
2. `TestIntegrationSpawnAgentWithCompanion` - now verifies ~30%/70% split
3. `TestIntegrationSpawnArchitectWithCompanion` - now verifies ~30%/70% split

Also fixed a pre-existing bug in `TestIntegrationNewManager` that referenced a non-existent `m.tmuxPath` field.

**Files changed:**
- `internal/tmux/integration_test.go` - Added width verification and helper function