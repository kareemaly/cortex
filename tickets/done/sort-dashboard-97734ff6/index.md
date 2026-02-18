---
id: 97734ff6-70a0-4d60-a7aa-dd939c2f6dfa
title: Sort dashboard sessions by start date (most recent first)
type: work
tags:
    - tui
created: 2026-02-12T12:33:08.019547Z
updated: 2026-02-12T13:04:30.5927Z
---
## Problem

The `cortex dashboard` (or projects list) does not sort sessions by their start date. Users want to see the most recently started sessions at the top for quick access.

## Context

Sessions already track `StartedAt` (time.Time, UTC) in the session struct. This field is set at creation time for all session types (architect, ticket, meta).

## Requirements

- Sort active sessions by `StartedAt` descending (most recently started at the top)
- This applies wherever sessions are listed in the dashboard/projects TUI

## Acceptance Criteria

- Sessions appear sorted with the most recently started at the top
- Sorting uses the existing `StartedAt` field
- No regressions in dashboard functionality