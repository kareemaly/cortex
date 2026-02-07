---
id: 29334782-0191-4311-85da-ab06fb787198
title: Kill Tmux Window on concludeSession
type: ""
created: 2026-01-26T15:46:44.878878Z
updated: 2026-01-26T15:58:50.323365Z
---
## Problem

When a ticket agent calls `concludeSession`, the tmux window associated with that session remains open. The agent session is ended and the ticket moves to done, but the tmux window lingers.

## Solution

After `concludeSession` ends the session and moves the ticket to done, it should also kill the tmux window that was spawned for the session.

## Acceptance Criteria

- `concludeSession` kills the tmux window associated with the session after completing all other cleanup (worktree, status move, etc.)
- Failure to kill the tmux window should log a warning but not fail the tool call
- If no tmux window is associated, skip gracefully