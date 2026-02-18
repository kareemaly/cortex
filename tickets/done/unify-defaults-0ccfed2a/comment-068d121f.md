---
id: 068d121f-5cd7-44fc-a3e5-5ef2490267af
author: claude
type: done
created: 2026-02-14T09:20:57.787948Z
---
## Summary

Unified the defaults directory structure and decoupled the `extend` field from config merging.

### What was done

1. **Collapsed defaults directories**: Merged `~/.cortex/defaults/claude-code/` and `~/.cortex/defaults/opencode/` (which had identical prompts) into a single `~/.cortex/defaults/main/` containing only prompt files.

2. **Self-contained project configs**: `cortex init` now generates a complete `cortex.yaml` with all agent settings inline (via `generateProjectConfig()`), instead of a minimal config that relied on recursive merging from the base.

3. **Removed config merging**: Deleted `MergeConfigs()` and the recursive `loadWithVisited()` config loader. The `extend` field now only stores a path for prompt resolution — no config values are inherited from it.

4. **Cleaned up dead code**: Removed `GetConfigDocs()`, `DefaultsDirForAgent()`, `CircularExtendError`, `merge.go`, and associated tests.

5. **Updated upgrade command**: `cortex defaults upgrade` now operates on `main/` only and cleans up legacy `claude-code/` and `opencode/` directories.

### Files changed
- 12 source files modified/deleted, 3 test files modified/deleted
- Net: +149 / -1,589 lines (significant simplification)
- Commit: 03dd628, pushed to origin/main