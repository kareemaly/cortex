---
id: 8bfac018-0358-44bc-ba7c-0f752e8171b2
author: claude
type: done
created: 2026-02-04T07:39:35.054576Z
---
## Notification Infrastructure: Config Schema + Tmux Attachment Detection

### Changes Made

**1. Notification Config Schema** (`internal/daemon/config/config.go`)
Added nested configuration structs for notification settings in global `settings.yaml`:

- `LocalChannelConfig`: Desktop notification settings (enabled, sound)
- `ChannelsConfig`: Container for channel configs (extensible for future channels like Slack, SMS)
- `BehaviorConfig`: Timing and batching settings
  - `batch_window_seconds: 60`
  - `notify_on_first_only: true`
  - `reminder_after_minutes: 10`
  - `suppress_when_attached: true`
- `EventsConfig`: Which events trigger notifications
  - `agent_waiting_permission: true`
  - `agent_idle: true`
  - `agent_error: true`
  - `ticket_review_requested: true`
- `NotificationsConfig`: Top-level container added to main `Config` struct

All defaults set in `DefaultConfig()`.

**2. Tmux Client Attachment Detection** (`internal/tmux/client.go`)
Added methods to existing `Manager` type:

- `ListClients(session)`: Returns all attached clients with TTY, session, window index, window name
- `IsUserAttached(session, windowName)`: Check if user viewing window by name
- `IsUserAttachedToWindow(session, windowIndex)`: Check by window index
- Both return `false` on error (safe default - don't suppress notifications if detection fails)

### Files Changed
- `internal/daemon/config/config.go` - Added 5 struct types + Notifications field
- `internal/daemon/config/config_test.go` - Added 3 tests for config behavior
- `internal/tmux/client.go` - New file with Client struct and methods
- `internal/tmux/client_test.go` - New file with 11 unit tests

### Verification
- All 136 tests pass (`make test`)
- Linter clean (`make lint` - 0 issues)

### Merge
Branch `ticket/notification` merged to main and pushed to origin.