---
id: 47f4810e-22e7-41bc-8eb2-ab99df61d696
title: 'Improve cortex defaults upgrade UX: diff colors and confirmation prompt'
type: work
tags:
    - tui
    - configuration
created: 2026-02-14T12:31:00.153259Z
updated: 2026-02-14T12:35:50.420772Z
---
## Problem

The `cortex defaults upgrade` command shows diffs when default prompt files have changed, but:

1. **Diff output lacks color** — additions and removals are not visually distinguished with red/green coloring, making it hard to see what actually changed
2. **Confirmation prompt is too subtle** — the yes/no question after showing the diff doesn't stand out, so users tend to just hit enter without reading. This is risky since it's updating prompt files that affect agent behavior.

## Requirements

- Add proper diff coloring: red for removals, green for additions (standard terminal diff colors)
- Make the confirmation prompt more visually prominent — use color, spacing, or formatting so it clearly stands out from the diff output
- Consider making the default answer "no" (require explicit "y") so users have to actively opt in, rather than passively hitting enter

## Acceptance Criteria

- Diff output shows red/green coloring for removals/additions
- Confirmation prompt is visually distinct from the diff content
- Users cannot accidentally accept changes by just pressing enter
- Build passes