---
id: 58b8d2d1-3d9b-411e-aa01-a60dcd6737dc
title: Remove "gd" keybinding from dashboard TUI
type: work
tags:
    - tui
    - cleanup
created: 2026-02-18T07:54:44.307033Z
updated: 2026-02-18T07:54:44.307033Z
---
## Problem

The dashboard TUI still has a "gd" keybinding referencing a CortexDaemon session concept that no longer exists (meta agent was removed).

## Requirements

- Remove the "gd" keybinding and any associated handler/logic from the dashboard TUI
- Remove it from help text / key hints if listed

## Acceptance Criteria

- "gd" no longer does anything in the dashboard
- No references to CortexDaemon session remain in dashboard keybindings