---
id: a2b2454b-496c-46aa-a4fd-ce1f766071e1
author: claude
type: comment
created: 2026-02-10T14:27:20.39591Z
---
## Root Cause Analysis

The bug is in how tmux resolves session names, not in the Go code's logic flow.

**Current approach**: `SessionExists("cortex")` runs `tmux has-session -t cortex:`. The trailing colon was intended to enforce exact matching.

**Why it fails**: The trailing colon is a session:window separator, NOT an exact-match enforcer. tmux still resolves the session part "cortex" using its default cascade: exact match → fnmatch → **substring matching**. Since "cortex" is a substring of "cortex-meta", tmux matches it. This causes:

1. `SessionExists("cortex")` → `true` (incorrectly matches `cortex-meta`)
2. `ListWindows("cortex")` → lists `cortex-meta`'s windows
3. On a prior corrupted spawn, `CreateArchitectWindow` renames `cortex-meta:0` from "meta" to "architect"
4. Subsequent runs: `WindowExists("cortex", "architect")` finds the renamed window → `StateActive` → "already spawned"

**Fix**: Use tmux's documented `=` prefix for exact matching. `tmux has-session -t =cortex` disables fnmatch and substring matching entirely. Apply `=` prefix to ALL tmux target specifications across the tmux package (22 target changes + 1 helper function).