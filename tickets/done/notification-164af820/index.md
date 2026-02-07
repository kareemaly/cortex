---
id: 164af820-a6cc-4177-8574-c5f9a51bbd30
title: Notification Dispatcher + Daemon Integration
type: work
created: 2026-02-04T07:49:47.000243Z
updated: 2026-02-04T08:23:03.078523Z
---
# Overview

Create the notification dispatcher that subscribes to the event bus, aggregates events, and routes to notification channels. Wire into daemon startup.

## Dependencies

This ticket builds on:
- **Ticket A**: Config schema in `internal/daemon/config/config.go` (NotificationsConfig)
- **Ticket B**: Channel interface and LocalChannel in `internal/notifications/`

## 1. Dispatcher Core

**Location:** `internal/notifications/dispatcher.go`

```go
type Dispatcher struct {
    config     *config.NotificationsConfig
    channels   []Channel
    tmux       TmuxChecker  // interface for IsUserAttached
    logger     *slog.Logger
    
    // State tracking
    mu              sync.Mutex
    pendingEvents   map[string][]Event  // projectPath → events in batch window
    attentionCount  map[string]int      // projectPath → agents needing attention
    batchTimers     map[string]*time.Timer
}

type TmuxChecker interface {
    IsUserAttached(session, windowName string) bool
}
```

### Event Handling Flow

1. **Subscribe to event bus** for all registered projects
2. **Filter events** based on config (agent_waiting_permission, agent_idle, agent_error, ticket_review_requested)
3. **Check attachment suppression** — skip if user attached to that agent's tmux window
4. **Add to batch window** — collect events for `batch_window_seconds`
5. **Track attention count** — count of agents in waiting_permission, error, or review states
6. **Apply notify_on_first_only** — only notify when going from 0→N agents needing attention
7. **Send batch notification** after window expires

### Events to Handle

| Event Type | Condition | Config Key |
|------------|-----------|------------|
| `SessionStatus` | status = `waiting_permission` | `agent_waiting_permission` |
| `SessionStatus` | status = `idle` | `agent_idle` |
| `SessionStatus` | status = `error` | `agent_error` |
| `ReviewRequested` | always | `ticket_review_requested` |

**Note:** For `SessionStatus` events, the status is in the event payload. Need to check `internal/events/bus.go` for payload structure.

## 2. Batch Window Logic

```go
func (d *Dispatcher) handleEvent(e events.Event) {
    if !d.shouldNotify(e) {
        return
    }
    
    // Check attachment suppression
    if d.config.Behavior.SuppressWhenAttached {
        if d.isUserAttached(e) {
            return
        }
    }
    
    d.mu.Lock()
    defer d.mu.Unlock()
    
    // Add to pending
    d.pendingEvents[e.ProjectPath] = append(d.pendingEvents[e.ProjectPath], e)
    
    // Update attention count
    prevCount := d.attentionCount[e.ProjectPath]
    d.attentionCount[e.ProjectPath] = d.calculateAttentionCount(e.ProjectPath)
    
    // Check notify_on_first_only
    if d.config.Behavior.NotifyOnFirstOnly && prevCount > 0 {
        return  // Already had agents waiting
    }
    
    // Start/reset batch timer
    d.resetBatchTimer(e.ProjectPath)
}

func (d *Dispatcher) sendBatchNotification(projectPath string) {
    d.mu.Lock()
    events := d.pendingEvents[projectPath]
    d.pendingEvents[projectPath] = nil
    d.mu.Unlock()
    
    if len(events) == 0 {
        return
    }
    
    notification := d.formatNotification(events)
    
    for _, ch := range d.channels {
        if ch.Available() {
            ch.Send(context.Background(), notification)
        }
    }
}
```

## 3. Message Formatting

**Single event:**
```
🔔 cortex: Agent waiting for plan approval
   ticket-abc in "cortex"
```

**Batched:**
```
🔔 cortex: 3 agents need attention
   • ticket-abc (plan approval)
   • ticket-def (review)
   • ticket-ghi (idle)
```

## 4. Re-nudge Timer (Optional)

If `reminder_after_minutes > 0` and agents still waiting after that time, send another notification.

## 5. Daemon Integration

**Location:** `cmd/cortexd/commands/serve.go`

```go
func runServe(cmd *cobra.Command, args []string) error {
    // ... existing setup ...
    
    // Initialize notification dispatcher
    var notifyDispatcher *notifications.Dispatcher
    if cfg.Notifications.Channels.Local.Enabled {
        localChannel := notifications.NewLocalChannel(
            cfg.Notifications.Channels.Local,
            logger,
        )
        notifyDispatcher = notifications.NewDispatcher(
            &cfg.Notifications,
            []notifications.Channel{localChannel},
            tmuxManager,
            logger,
        )
        
        // Subscribe to all registered projects
        for _, projectPath := range cfg.Projects {
            go notifyDispatcher.Subscribe(ctx, bus, projectPath)
        }
    }
    
    // ... rest of serve ...
}
```

## 6. CLI Test Command

**Location:** `cmd/cortex/commands/notify.go`

```bash
cortex notify test              # Send test notification via all enabled channels
cortex notify test --sound      # With sound
cortex notify test --message "Custom message"
```

## Files to Create/Modify

- `internal/notifications/dispatcher.go` — Dispatcher implementation
- `internal/notifications/dispatcher_test.go` — Unit tests
- `cmd/cortexd/commands/serve.go` — Wire dispatcher into daemon
- `cmd/cortex/commands/notify.go` — CLI test command
- `cmd/cortex/commands/root.go` — Register notify command

## Testing

- Unit tests for event filtering logic
- Unit tests for batch window aggregation
- Unit tests for attention count tracking
- Unit tests for notify_on_first_only behavior
- Unit tests for attachment suppression
- Integration test: emit event → notification sent

## Out of Scope

- ntfy channel
- Webhook channel
- Reminders feature (future ticket)