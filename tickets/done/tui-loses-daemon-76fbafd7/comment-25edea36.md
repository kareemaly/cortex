---
id: 25edea36-0d0c-4e0b-919e-899a3f055e67
author: claude
type: comment
created: 2026-02-05T10:28:41.077978Z
---
## Root Cause Analysis

**The bug**: When viewing a ticket in TUI and pressing "d" to open lazygit in a tmux popup, the TUI shows "failed to connect to daemon: context" error.

**Root cause identified**: The daemon's HTTP handler blocks on `tmux display-popup` execution.

### Call Flow

1. TUI presses "d" → calls `executeDiffAction()` (model.go:576-584)
2. SDK sends HTTP POST to `/tickets/{id}/comments/{commentID}/execute` with **10-second timeout**
3. Daemon handler `ExecuteAction` (tickets.go:808-923) calls `h.deps.TmuxManager.DisplayPopup()` (line 913)
4. `DisplayPopup()` (popup.go:18) calls `m.run(args...)` 
5. `run()` (tmux.go:62-64) calls `m.runner.Run(args...)`
6. `execRunner.Run()` (tmux.go:22-25) uses `cmd.CombinedOutput()` which **blocks until the popup closes**
7. After 10 seconds, the TUI's HTTP client times out → "context deadline exceeded"

### Why `tmux display-popup -E` blocks

The `-E` flag tells tmux to wait for the command to exit. Combined with Go's `cmd.CombinedOutput()`, the daemon handler blocks until the user closes lazygit. This causes the HTTP request to timeout.

### Evidence

- `popup.go:18`: `_, err := m.run(args...)`  
- `tmux.go:23-24`: `cmd := exec.Command(r.tmuxPath, args...)` + `return cmd.CombinedOutput()`
- `client.go:34-36`: HTTP client with `Timeout: 10 * time.Second`

The error message "context" is truncated from "context deadline exceeded".