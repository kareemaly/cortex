---
id: fec3eb58-2289-4a0f-9e7b-8f5c3fb84f74
title: 'Research: Architect tmux pane split is 50/50 instead of 30/70'
type: research
tags:
    - tmux
    - architect
    - research
created: 2026-02-13T09:25:34.718434Z
updated: 2026-02-13T10:20:01.452708Z
---
## Problem

Architect session tmux windows consistently open with a 50/50 pane split (agent pane / companion pane), but they should match ticket agent windows which use a 30/70 split.

## Prior Investigation

Last session this was briefly explored. The 30/70 split code was found to be present and correct, and `ResetWindowPanes` cleanup exists for architect sessions. The hypothesis was "stale tmux session" — but the problem persists across fresh spawns, so something else is going on.

## Research Goals

1. Trace the exact code path for architect session tmux window creation — how does it differ from ticket agent session creation?
2. Identify where the pane split percentage is set for architect vs ticket agent windows
3. Determine why the 30/70 split isn't being applied to architect windows despite the code appearing correct
4. Document findings with specific file paths and line numbers

## Acceptance Criteria

- Root cause identified and documented
- Clear explanation of the code path difference between architect and ticket agent pane splits