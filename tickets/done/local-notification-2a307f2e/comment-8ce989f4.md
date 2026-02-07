---
id: 8ce989f4-d543-475a-8698-e34ae52336c9
author: claude
type: done
created: 2026-02-04T07:37:38.279025Z
---
## Summary

Implemented the local desktop notification channel for macOS and Linux.

### Files Created

1. **`internal/notifications/channel.go`**
   - `Notification` struct with Title, Body, Sound, Urgency fields
   - `Channel` interface defining `Name()`, `Send()`, `Available()` methods

2. **`internal/notifications/local.go`**
   - `CommandRunner` interface for testability (follows tmux.go pattern)
   - `LocalChannel` implementation with platform detection via `runtime.GOOS`
   - macOS: terminal-notifier (primary), osascript (fallback)
   - Linux: notify-send with urgency mapping
   - Graceful degradation when no tools available (returns nil, logs warning)
   - `sync.Once` for one-time tool detection logging

3. **`internal/notifications/local_test.go`**
   - 13 unit tests with mock CommandRunner
   - Tests cover all platforms, sound support, urgency mapping, graceful degradation

### Verification
- All tests pass
- Linter passes with 0 issues
- Merged to main and pushed to origin