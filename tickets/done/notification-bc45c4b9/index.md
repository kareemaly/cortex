---
id: bc45c4b9-ec64-4f91-83ae-4444e11b668c
title: 'Notification Infrastructure: Config Schema + Tmux Attachment Detection'
type: work
created: 2026-02-04T07:27:46.753844Z
updated: 2026-02-04T07:39:35.056592Z
---
# Overview

Add the foundational infrastructure for the notification system: configuration schema and tmux user attachment detection.

## 1. Notification Config Schema

Add `notifications` section to global settings (`~/.cortex/settings.yaml`).

**Location:** `internal/daemon/config/config.go`

```yaml
notifications:
  channels:
    local:
      enabled: true                    # Default: true
      sound: true                      # Play sound with notification
      
  behavior:
    batch_window_seconds: 60           # Collect events before sending
    notify_on_first_only: true         # Only notify on 0→N transition
    reminder_after_minutes: 10         # Re-nudge if still waiting (0 = disabled)
    suppress_when_attached: true       # Skip notification if user is in that tmux session
    
  events:
    agent_waiting_permission: true     # Plan approval needed
    agent_idle: true                   # Agent finished working
    agent_error: true                  # Agent hit error state
    ticket_review_requested: true      # Agent called requestReview
```

**Requirements:**
- Add `NotificationsConfig` struct with nested `ChannelsConfig`, `BehaviorConfig`, `EventsConfig`
- Set sensible defaults (local enabled, others as shown above)
- Load and validate on daemon startup
- Unit tests for config loading and defaults

## 2. Tmux Client Attachment Detection

Extend tmux manager to detect if a user is attached to a session/window.

**Location:** `internal/tmux/`

**New methods:**
- `ListClients(session string) ([]Client, error)` — wraps `tmux list-clients -t SESSION`
- `IsUserAttached(session, windowName string) bool` — returns true if any client is viewing the specified window

**Client struct:**
```go
type Client struct {
    TTY         string  // e.g., /dev/ttys001
    Session     string  // Session name
    Window      int     // Active window index
    WindowName  string  // Active window name (if available)
}
```

**Requirements:**
- Parse `tmux list-clients -t SESSION -F "#{client_tty}:#{window_index}:#{window_name}"` output
- Handle case where no clients attached (empty list, not error)
- Unit tests with mocked tmux output

## Files to Modify/Create

- `internal/daemon/config/config.go` — add NotificationsConfig
- `internal/daemon/config/config_test.go` — test config loading
- `internal/tmux/client.go` (new) — Client struct and ListClients
- `internal/tmux/manager.go` — add IsUserAttached helper
- `internal/tmux/client_test.go` (new) — unit tests

## Out of Scope

- Notification dispatcher logic (Ticket C)
- Notification channel implementations (Ticket B)
- Actually sending notifications