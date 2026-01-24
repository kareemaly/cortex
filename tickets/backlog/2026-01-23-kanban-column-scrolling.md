# Kanban Column Scrolling

## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Problem

When a column (especially Done) has many tickets, the entire kanban UI overflows the terminal height and columns become invisible.

## Requirements

- Columns should have a max height based on terminal height
- Scroll within columns when navigating with j/k
- Add vim shortcuts:
  - `ctrl+u` / `ctrl+d` - scroll 10 tickets
  - `gg` - jump to first ticket
  - `G` - jump to last ticket
