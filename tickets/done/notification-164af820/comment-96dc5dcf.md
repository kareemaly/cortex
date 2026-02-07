---
id: 96dc5dcf-8687-464f-8f18-eaba03fa9397
author: claude
type: review_requested
created: 2026-02-04T08:07:15.547509Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/d81ea268-15f7-40f7-8fe1-13d8d1bba81f
---
## Notification Dispatcher Implementation

### Summary
Implemented the notification dispatcher system that subscribes to the event bus, aggregates events within a batch window, tracks "attention count" (agents needing attention), and routes notifications to desktop notification channels.

### Files Created

1. **`internal/notifications/dispatcher.go`** - Core dispatcher logic with:
   - `Dispatcher` struct managing subscriptions, attention tracking, and batching
   - Event classification for `SessionStatus` and `CommentAdded` events
   - Mapping to notifiable event types: `agent_waiting_permission`, `agent_idle`, `agent_error`, `review_requested`
   - Config-based filtering for enabled/disabled event types
   - Attachment suppression via `tmuxManager.IsUserAttached()`
   - Global attention tracking across all projects for `notify_on_first_only`
   - Batch window with configurable duration (0 = immediate flush)
   - Message formatting for single and batched notifications

2. **`internal/notifications/dispatcher_test.go`** - Comprehensive unit tests covering:
   - Event classification for all session statuses
   - CommentAdded review detection
   - Config filtering
   - Attention tracking (wasZero detection, clear, count)
   - Batch window behavior
   - `notify_on_first_only` mode
   - Notification formatting (single and batch)
   - Urgency levels per event type
   - Graceful shutdown

3. **`cmd/cortex/commands/notify.go`** - CLI test command:
   - `cortex notify test` - Send test notification
   - `cortex notify test --sound` - With sound
   - `cortex notify test -m "Custom"` - Custom message

### Files Modified

4. **`cmd/cortexd/commands/serve.go`** - Daemon integration:
   - Creates `LocalChannel` and checks availability
   - Creates `Dispatcher` with config, channels, store manager, tmux manager, and event bus
   - Subscribes to all registered projects
   - Graceful shutdown on server exit

### Verification
- `make build` - Compiles successfully
- `make lint` - No issues
- `make test` - All tests pass (including 16 new dispatcher tests)