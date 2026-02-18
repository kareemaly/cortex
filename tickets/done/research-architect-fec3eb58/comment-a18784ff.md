---
id: a18784ff-60b4-450d-8032-efccaf4d0f94
author: claude
type: review_requested
created: 2026-02-13T09:36:14.564368Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Research Complete: Architect Tmux Pane Split 50/50 Instead of 30/70

### Root Cause (Empirically Confirmed)

**Tmux's window resize algorithm distributes extra space evenly across panes, not proportionally.** When `SpawnArchitect` creates a new detached session, the default window size is 80x24. The `-p 70` split correctly creates 30/70 panes at that size. But when the CLI later attaches (`tmux attach-session`), the window resizes to the terminal width (e.g., 200+ columns), and tmux distributes the extra columns ~50/50 between panes.

Empirical proof:
- Split at 80 cols → resize to 200: **41.7% / 58.3%** (broken)
- Split at 200 cols directly: **29.6% / 70.4%** (correct)
- At 300 cols: ratio drifts to **44.5% / 55.5%** (approaching 50/50)

### Why Ticket Agents Are Unaffected

When `SpawnAgent` is called, the tmux session already has an attached client (the architect is running). New windows inherit the attached client's terminal size, so the split happens at the correct width from the start — no resize needed.

### Critical Code Location

`internal/tmux/session.go:41` — `CreateSession` uses `new-session -d` without `-x`/`-y` flags, resulting in a default 80x24 window.

### Why Previous Investigations Missed This

Two prior tickets (`bd84ba87`, `680d88da`) investigated this issue. Both concluded the code was correct because integration tests verified the 30/70 ratio — but the tests only check the detached session, never testing what happens after client attachment triggers a window resize.

### Recommended Fix

**Option A (Recommended)**: Pass the CLI's terminal dimensions in the spawn request. The daemon forwards them to `tmux new-session -d -x <width> -y <height>`, so the session starts at the correct size and the split produces the right ratio from the start.

**Option B**: Set a tmux `after-client-attached` hook that resizes panes to 70% on attach. Fewer code changes but relies on tmux hook support.

### Findings Doc

Created: "Root Cause: Architect Tmux Pane Split 50/50 Instead of 30/70" in docs/findings/ with full code path analysis, empirical data tables, and implementation details for both fix options.