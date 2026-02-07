---
id: a1f6c8e1-fa63-4aae-a62c-a40d382fb05a
title: Fix staticcheck nil pointer warning in dispatcher test
type: work
created: 2026-02-04T13:17:26.688647Z
updated: 2026-02-04T13:19:52.952871Z
---
## Issue

CI lint failing with staticcheck SA5011:

```
internal/notifications/dispatcher_test.go:261:5: SA5011: this check suggests that the pointer can be nil
internal/notifications/dispatcher_test.go:264:16: SA5011: possible nil pointer dereference
```

The code checks `if notifiable == nil` then immediately dereferences `notifiable.Type`.

## Fix

Restructure the nil check to return/continue before dereferencing, or use else block properly.