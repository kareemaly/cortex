---
id: b8406ddc-208a-4eee-bc01-720d04ae9c61
title: Limit Done Tickets in Architect Prompt to Latest 10
type: ""
created: 2026-01-26T15:41:02.235348Z
updated: 2026-01-26T15:45:53.971603Z
---
## Problem

The architect prompt currently includes ALL done tickets when rendering the ticket list context. As the project accumulates completed tickets, this wastes context window space with irrelevant historical tickets.

## Solution

When building the architect prompt, limit the "Done" section to only the 10 most recently updated tickets. Backlog, in-progress, and review tickets should continue to show all entries.

## Acceptance Criteria

- Done section in architect prompt shows at most 10 tickets, sorted by most recently updated
- Other status sections (backlog, progress, review) remain unchanged
- If there are fewer than 10 done tickets, show all of them