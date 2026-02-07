---
id: 11f2f7f5-9030-41b2-8379-291d6db6ed15
author: claude
type: review_requested
created: 2026-02-04T07:38:44.364772Z
action:
    type: git_diff
    args:
        commit: 6fd875b
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/8daff18f-0b1f-41e0-b462-b08bd3d7ed39
---
## Summary

Added foundational notification infrastructure with two components:

### 1. Notification Config Schema (`internal/daemon/config/config.go`)

Added nested config structs for notification settings in global `settings.yaml`:

- **LocalChannelConfig**: Desktop notification settings (enabled, sound)
- **ChannelsConfig**: Container for channel configs (extensible for future channels)
- **BehaviorConfig**: Timing and batching settings
  - `batch_window_seconds: 60` - Batch notifications within window
  - `notify_on_first_only: true` - Only notify on first event in batch
  - `reminder_after_minutes: 10` - Reminder for unaddressed events
  - `suppress_when_attached: true` - Suppress if user viewing window
- **EventsConfig**: Which events trigger notifications (all default to true)
  - `agent_waiting_permission`
  - `agent_idle`
  - `agent_error`
  - `ticket_review_requested`
- **NotificationsConfig**: Top-level container

All defaults are set in `DefaultConfig()` and tested for round-trip persistence.

### 2. Tmux Client Attachment Detection (`internal/tmux/client.go`)

Added methods to the existing `Manager` type:

- **ListClients(session)**: Returns all attached clients with TTY, session, window index, and window name. Handles colons in window names with `SplitN(..., 4)`.
- **IsUserAttached(session, windowName)**: Returns true if any client is viewing the specified window by name.
- **IsUserAttachedToWindow(session, windowIndex)**: Same but matches by index.

Both `IsUserAttached*` methods return `false` on error (safe default for notifications - don't suppress if we can't determine attachment).

### Tests Added

- `TestDefaultConfigNotifications` - Verify all default values
- `TestNotificationsConfigRoundTrip` - Save/load with modified values
- `TestNotificationsPartialYAML` - Partial YAML preserves defaults for unset fields
- 11 tests for tmux client detection covering parsing, empty output, session not found, colons in window names, and IsUserAttached behavior

### Verification

```
make test   # All 136 tests pass
make lint   # 0 issues
```