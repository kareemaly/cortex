---
id: e5e07b00-f6a2-4b4c-b132-1d87dd54dee2
title: Add Reject Flow with REJECT.md Prompt Template
type: work
created: 2026-02-02T14:15:47.270425Z
updated: 2026-02-02T15:41:44.997667Z
---
## Summary

Implement reject flow mirroring approve flow. When rejecting, send REJECT.md instructions to the agent telling it to cleanly rollback all changes and conclude the session.

## Requirements

### Prompt Template
- Create `REJECT.md` in `internal/install/defaults/claude-code/prompts/ticket/work/`
- Instructions should be concise and concrete for any AI agent:
  - Git stash all uncommitted changes
  - Rollback/revert any commits made during this session
  - Remove any environment changes outside git (temp files, env vars, etc.)
  - Conclude the session cleanly
- Keep it short and actionable — agent should execute immediately

### API Endpoint
- `POST /tickets/{id}/sessions/{session_id}/reject`
- Renders REJECT.md template (similar to approve flow)
- Sends to agent pane via tmux
- Focuses tmux window
- Ticket moves to `done` after agent concludes (via concludeSession)

### TUI Integration
- `x` key in review comment modal triggers reject
- No feedback input needed — just confirm action
- Calls reject endpoint
- Focuses tmux window

## Flow
1. User presses `x` on review comment
2. TUI calls reject endpoint
3. Daemon renders REJECT.md and sends to agent pane
4. Agent reads instructions, rolls back changes, calls `concludeSession`
5. Ticket moves to done

## Acceptance Criteria
- [ ] REJECT.md template exists with rollback instructions
- [ ] Reject endpoint sends template to agent pane
- [ ] Agent receives clear rollback instructions
- [ ] TUI `x` key triggers reject (no input prompt)
- [ ] Session concludes after rollback