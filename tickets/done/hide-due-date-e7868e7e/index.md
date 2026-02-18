---
id: e7868e7e-3919-4698-8ebc-acadacf7a64f
title: Hide due date badges on done tickets in kanban TUI
type: work
tags:
    - tui
    - cleanup
created: 2026-02-12T11:26:44.246803Z
updated: 2026-02-12T11:28:50.39948Z
---
## Problem

Tickets in the "done" column still show "OVERDUE" and "DUE SOON" badges, which is misleading — once a ticket is done, the due date is no longer relevant.

## Acceptance Criteria

- "OVERDUE" and "DUE SOON" badges are only shown on tickets in the **backlog** column
- Done (and progress/review) tickets do not display due date badges
- No other visual regressions on ticket cards