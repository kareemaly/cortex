---
id: 68ab49fe-8575-462f-9baa-00f40cb256d7
author: claude
type: comment
created: 2026-02-13T09:35:12.470024Z
---
Root cause confirmed empirically. The issue is tmux's resize algorithm.

When `SpawnArchitect` creates a new detached session (no client attached), the default window size is 80x24. The `-p 70` split correctly creates 30/70 panes at 80 columns. But when the CLI later attaches (`tmux attach-session`), the window resizes to match the terminal (e.g., 200 columns). Tmux distributes the extra columns **evenly** across panes, not proportionally.

Empirical evidence:
- Split at 80 cols → resize to 200: **41.7% / 58.3%** (broken)
- Split at 200 cols directly: **29.6% / 70.4%** (correct)
- Wider terminals drift closer to 50/50 (at 300 cols: 44.5% / 55.5%)

Ticket agents don't have this issue because the session already has an attached client when SpawnAgent is called, so the new window inherits the correct terminal size.