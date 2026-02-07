---
id: c0c08ddf-5ff2-493d-abf7-1242618a9298
author: claude
type: done
created: 2026-02-04T08:23:03.076517Z
---
## Summary

Implemented notification dispatcher system for event-driven desktop notifications.

### Changes Made

**Created Files:**
- `internal/notifications/dispatcher.go` - Core dispatcher with:
  - Event bus subscription per project
  - Event classification (SessionStatus → agent states, CommentAdded → review requests)
  - Config-based filtering for enabled event types
  - Attachment suppression via tmux window detection
  - Global attention tracking for notify_on_first_only mode
  - Configurable batch window (0 = immediate flush)
  - Single and batched notification formatting

- `internal/notifications/dispatcher_test.go` - 16 unit tests covering event classification, config filtering, attention tracking, batching, formatting, and graceful shutdown

- `cmd/cortex/commands/notify.go` - CLI test command (`cortex notify test`)

**Modified Files:**
- `cmd/cortexd/commands/serve.go` - Wired dispatcher into daemon startup with graceful shutdown

### Verification
- Build: ✅ Passes
- Lint: ✅ No issues  
- Tests: ✅ All pass

### Commit
`53c28c0` - feat(notifications): add dispatcher for event-driven desktop notifications

Merged to main and pushed to origin.