---
id: e7ea8241-add6-4723-a4d7-0623cb586915
author: claude
type: comment
created: 2026-02-10T07:14:01.526159Z
---
## Root Cause Analysis

**Primary bug:** `SessionExists` in `internal/tmux/session.go:17` uses `has-session -t name` without a trailing colon. Tmux interprets bare names as prefix-matchable targets, so `has-session -t cortex` matches `cortex-meta`.

**Evidence:** The codebase already documents this behavior in `internal/tmux/window.go:45-48`:
```
// Use "session:" (with trailing colon) to force tmux to resolve the target as a session name.
// Without the colon, tmux treats -t as a target-window and may prefix-match a window name
```

`CreateWindow` was fixed but `SessionExists` was not. Since `SessionExists` is the guard for 9+ other functions (CreateSession, KillSession, AttachSession, CreateWindow, ListWindows, SpawnArchitect, etc.), the bug cascades.

**Full audit of bare session targets in `internal/tmux/`:**
| Function | File:Line | Command | Vulnerable? |
|---|---|---|---|
| `SessionExists` | session.go:17 | `has-session -t name` | **YES - primary bug** |
| `KillSession` | session.go:63 | `kill-session -t name` | **YES** |
| `AttachSession` | session.go:88 | `switch-client -t name` | Low (interactive) |
| `AttachSession` | session.go:90 | `attach-session -t name` | Low (interactive) |
| `SwitchClient` | session.go:96 | `switch-client -t session` | **YES** (daemon-side) |
| `ListWindows` | window.go:157 | `list-windows -t session` | Guarded by SessionExists |
| `ListClients` | client.go:30 | `list-clients -t session` | Guarded by SessionExists |
| `DisplayPopup` | popup.go:12 | `display-popup -t session` | Low (interactive) |

All other `-t` usages already use `session:index` or `session:index.pane` format (safe).