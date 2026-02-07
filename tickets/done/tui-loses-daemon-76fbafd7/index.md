---
id: 76fbafd7-f737-4858-ab77-120304c2bd20
title: TUI loses daemon connection when tmux popup opens
type: debug
created: 2026-02-05T10:23:19.745853Z
updated: 2026-02-05T10:32:41.02841Z
---
## Bug Description

When viewing a ticket in the TUI (`cortex show`) and opening a tmux popup (e.g., pressing "d" for lazygit), the TUI in the background loses its daemon connection and shows an error.

## Error Message
```
Error: failed to connect to daemon: Post "http://localhost:4200/tickets/{id}/comments/{comment-id}/execute": context

Press [r] to retry or [q] to quit

Is the daemon running? Start it with: cortexd start
```

## Steps to Reproduce
1. Open a ticket in review: `cortex show <ticket-id>`
2. Press "d" to open lazygit in a tmux popup
3. Observe the TUI in the background shows the connection error
4. Press "r" to retry — it works again

## Expected Behavior
TUI should maintain daemon connection or gracefully handle temporary disconnection without user intervention.

## Likely Cause
The error message shows `context` truncated — suggests HTTP request context is being cancelled when:
- Focus changes to popup
- TUI loses foreground state
- Some polling/refresh operation times out

## Areas to Investigate
- `internal/cli/tui/` — How does the TUI handle background state?
- `internal/cli/sdk/client.go` — HTTP client timeout/context settings
- Any polling or refresh loops that might fail when TUI is backgrounded

## Acceptance Criteria
- TUI handles being backgrounded gracefully
- No manual retry needed when returning from tmux popup