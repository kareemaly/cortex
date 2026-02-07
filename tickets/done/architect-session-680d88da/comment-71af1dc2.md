---
id: 71af1dc2-4ce1-4425-97a5-83078049c0d1
author: claude
type: done
created: 2026-02-05T12:37:32.477862Z
---
## Summary

Investigated the architect session split pane layout and verified the 30%/70% split was already implemented correctly.

### Key Findings

1. **Code path traced**: `cortex architect` → `POST /architect/spawn` → `spawner.Spawn()` → `SpawnArchitect()` → `SplitWindowHorizontal()`
2. **Implementation is correct**: `internal/tmux/pane.go:12` uses `tmux split-window -h -p 70` which gives:
   - 30% to left pane (agent)
   - 70% to right pane (companion/kanban)
3. **Manual verification confirmed** the tmux `-p 70` flag produces the expected ratio

### Changes Made

Added pane width verification to integration tests to catch future regressions:

**File: `internal/tmux/integration_test.go`**
- Added `parsePaneWidths()` helper function to parse tmux list-panes output
- Updated `TestIntegrationSplitWindowHorizontal` to verify ~30%/70% split
- Updated `TestIntegrationSpawnAgentWithCompanion` to verify ~30%/70% split  
- Updated `TestIntegrationSpawnArchitectWithCompanion` to verify ~30%/70% split
- Fixed pre-existing bug in `TestIntegrationNewManager` that referenced non-existent `m.tmuxPath` field

### Conclusion

The implementation was already correct. If users see 50%/50% splits, it's likely from old sessions created before this code was in place. Killing and recreating the session will use the correct 30%/70% layout.