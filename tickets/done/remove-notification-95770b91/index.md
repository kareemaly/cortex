---
id: 95770b91-a280-407a-b449-caaf289ca112
title: Remove notification system entirely
type: chore
tags:
    - cleanup
    - notifications
created: 2026-02-07T13:00:10.86227Z
updated: 2026-02-07T13:11:07.284054Z
---
## Context

The OS desktop notification system (`internal/notifications/`) is not providing value. It's well-architected but unused in practice. Remove it completely.

## What to remove

### 1. Delete `internal/notifications/` directory (5 files)
- `channel.go` — Channel interface + Notification struct
- `local.go` — LocalChannel (terminal-notifier, osascript, notify-send)
- `local_test.go`
- `dispatcher.go` — Event-to-notification router with batching/dedup
- `dispatcher_test.go`

### 2. Remove from daemon startup (`cmd/cortexd/commands/serve.go`)
- Delete dispatcher creation block (~lines 91-111)
- Delete dispatcher shutdown block (~lines 128-131)
- Remove `notifications` import

### 3. Delete CLI command (`cmd/cortex/commands/notify.go`)
- Entire file — `cortex notify test` command
- Remove from command registration in parent command

### 4. Clean up config (`internal/daemon/config/config.go`)
- Remove structs: `NotificationsConfig`, `ChannelsConfig`, `LocalChannelConfig`, `BehaviorConfig`, `EventsConfig`
- Remove `Notifications` field from `Config` struct
- Remove from `DefaultConfig()`
- Update any config tests that reference notifications

### 5. Documentation
- Remove `internal/notifications/` from Key Paths table in CLAUDE.md

## What NOT to remove
- Event bus (`internal/events/`) — used by SSE and other systems
- `IsUserAttached()`/`IsUserAttachedToWindow()` in `internal/tmux/client.go` — may be useful elsewhere, and the methods are on the tmux client not the notification system

## Acceptance criteria
1. `internal/notifications/` directory is gone
2. Daemon starts and runs without notification code
3. `cortex notify` command is removed
4. Config schema no longer has notification fields
5. `make build && make lint && make test` pass
6. No dead imports or references remain