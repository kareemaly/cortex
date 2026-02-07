---
id: c065b195-701d-4479-8a9e-17bca1860eac
title: Sort Kanban Tickets by Created Date Instead of Updated
type: ""
created: 2026-01-27T10:07:18.876558Z
updated: 2026-01-27T10:11:09.803089Z
---
## Problem

Kanban columns currently sort tickets by `updated_at`. When agents are actively working on tickets, the updated timestamp changes constantly, causing tickets to shuffle around in the progress column. This is distracting and makes it hard to track tickets visually.

## Solution

Sort tickets by `created_at` instead of `updated_at`, with most recently created tickets first. The created date is stable and won't change as agents work.

## Scope

- Identify where ticket sorting happens (likely in the API response or TUI model) and change the sort key from `updated_at` to `created_at`, descending (newest first)

## Acceptance Criteria

- [ ] Tickets in all kanban columns are sorted by created date (newest first)
- [ ] Ticket order remains stable while agents are working
- [ ] No change to the date displayed on the card metadata line