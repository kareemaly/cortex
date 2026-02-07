---
id: 0bbab9c7-c56a-47fe-8487-940d70b43d2f
title: Spawn Pane Split Ratio
type: ""
created: 2026-01-24T09:52:43Z
updated: 2026-01-24T09:52:43Z
---
## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Problem

Current tmux split gives equal space to AI and companion panes.

## Requirements

- Split 30% for AI pane, 70% for companion pane (kanban/ticket detail)

## Implementation

### Commits
- `e792821` feat: set spawn pane split ratio to 30/70

### Key Files Changed
- `internal/tmux/pane.go` - Added `-p 70` flag to `split-window` command

### Changes
- Modified `SplitWindowHorizontal` to pass `-p 70` flag to tmux, giving the new (right) pane 70% of the horizontal space