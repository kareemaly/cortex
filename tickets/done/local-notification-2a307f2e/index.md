---
id: 2a307f2e-1e9f-40da-be52-c5abc4e9a1d8
title: Local Notification Channel Implementation
type: work
created: 2026-02-04T07:27:47.012674Z
updated: 2026-02-04T07:37:38.281017Z
---
# Overview

Implement the local desktop notification channel for macOS and Linux.

## 1. Channel Interface

Define a generic channel interface that all notification channels implement.

**Location:** `internal/notifications/channel.go`

```go
type Notification struct {
    Title   string
    Body    string
    Sound   bool
    Urgency string  // "low", "normal", "critical" (maps to platform equivalents)
}

type Channel interface {
    Name() string
    Send(ctx context.Context, n Notification) error
    Available() bool  // Check if channel can be used (tools installed, etc.)
}
```

## 2. Local Channel Implementation

**Location:** `internal/notifications/local.go`

### macOS Support
1. Try `terminal-notifier` first (better UX, supports sounds)
   ```bash
   terminal-notifier -title "Title" -message "Body" -sound default
   ```
2. Fall back to `osascript` if terminal-notifier not installed
   ```bash
   osascript -e 'display notification "Body" with title "Title" sound name "default"'
   ```

### Linux Support
Use `notify-send` (libnotify):
```bash
notify-send -u normal "Title" "Body"
```

### Platform Detection
- Use `runtime.GOOS` to detect platform
- Check tool availability with `exec.LookPath()`
- Log warning if no notification tool available, but don't crash

### Sound Support
- macOS terminal-notifier: `-sound default`
- macOS osascript: `sound name "default"`
- Linux: no standard sound support in notify-send (note in docs)

## 3. Graceful Degradation

- If no notification tools available, `Available()` returns false
- `Send()` should log and return nil (not error) if tools missing — daemon shouldn't crash
- Log which tool is being used on first notification

## Files to Create

- `internal/notifications/channel.go` — Channel interface, Notification struct
- `internal/notifications/local.go` — LocalChannel implementation
- `internal/notifications/local_test.go` — unit tests with mocked exec

## Testing

- Mock `exec.Command` for unit tests
- Test platform detection logic
- Test fallback behavior (terminal-notifier missing → osascript)
- Test graceful degradation when no tools available

## Out of Scope

- ntfy channel (future)
- Webhook channel (future)
- Dispatcher logic (Ticket C)
- Configuration loading (Ticket A)