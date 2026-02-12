---
id: 3943db26-72a1-44fe-bf36-b17879b00f35
title: Inject ticket comments into agent KICKOFF prompt
type: work
tags:
    - mcp
    - architect
    - session
references:
    - ticket:06a6c1c8-19da-4fea-b03a-2872676e20f7
created: 2026-02-10T09:40:30.902638Z
updated: 2026-02-10T09:52:08.025831Z
---
## Problem

The ticket agent no longer has a `readTicket` tool (removed in the recent tool optimization — ticket 06a6c1c8). This means the agent cannot see comments on its own ticket, including blocker notes, decision context, and prior investigation findings.

## Requirements

- When composing the KICKOFF prompt for a ticket agent session, include the ticket's comments in the injected context
- Comments should be formatted clearly with their type (comment, blocker, review_requested, done) and timestamp
- This is especially important for `resume` and `fresh` mode spawns where prior comments contain critical context from previous attempts

## Acceptance Criteria

- Ticket agent sees all existing comments on its ticket at session start
- Comment type and timestamp are visible in the injected prompt
- Works for all spawn modes (normal, resume, fresh)