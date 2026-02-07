---
id: b113b4e9-c413-4e0c-947d-0bddc17a40e7
title: 'Fix kanban column overflow: replace lipgloss Height with viewport'
type: debug
created: 2026-02-06T09:31:25.624444Z
updated: 2026-02-06T09:46:22.170659Z
---
## Problem

When focused on the "done" column in `cortex kanban` and there are many tickets with long titles (2+ lines), the entire terminal UI scrolls instead of just the column content.

## Root Cause

Kanban columns use lipgloss's `Height()` to constrain column height (`column.go` lines 262-264), but lipgloss doesn't clip overflowing content — it's a styling method, not a viewport. When rendered ticket content exceeds the height constraint, it leaks out and pushes the parent layout.

This is a known lipgloss limitation (see charmbracelet/lipgloss#528).

## Fix

Replace the manual scroll logic + lipgloss `Height()` with a `bubbles/viewport` component per column. The viewport properly clips content within a fixed height.

### Changes needed

1. **`internal/cli/tui/kanban/column.go`**:
   - Add a `viewport.Model` field to the `Column` struct
   - In `Column.View()`: render all visible tickets into the viewport instead of applying `Height()` style
   - Update `MoveUp/MoveDown/ScrollUp/ScrollDown` to work with viewport position instead of manual `scrollOffset`

2. **`internal/cli/tui/kanban/model.go`**:
   - Initialize viewports when columns are created
   - Handle viewport dimension updates on resize
   - Route viewport updates through the `Update()` method

### Precedent

The ticket detail TUI (`internal/cli/tui/ticket/model.go`) already uses `bubbles/viewport` successfully for scrollable content.