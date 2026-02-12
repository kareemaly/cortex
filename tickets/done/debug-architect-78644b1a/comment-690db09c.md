---
id: 690db09c-1886-4ac4-ae03-293e151454a7
author: claude
type: done
created: 2026-02-10T14:52:24.866144Z
---
## Root Cause

tmux resolves `-t name:` targets using a cascade: exact match → fnmatch → substring matching. The trailing colon (`name+":"`) only separates session:window — it does NOT disable substring matching for the session name part. When a project is named "cortex", `tmux has-session -t cortex:` resolves "cortex" as a substring of the existing "cortex-meta" session and returns success. This causes `SessionExists("cortex")` → true → `ListWindows` queries cortex-meta's windows → false "already spawned" detection.

## Resolution

Added a `sessionTarget(name string)` helper in `internal/tmux/tmux.go` that returns `"=" + name`, leveraging tmux's documented `=` prefix for exact name matching (supported since tmux 2.1, 2015). `tmux has-session -t =cortex:` only matches a session named exactly "cortex", not "cortex-meta".

Applied the helper consistently across all 22 tmux target constructions in 7 source files:
- `tmux.go` — added `sessionTarget()` helper
- `session.go` — 6 targets (SessionExists, KillSession, AttachSession ×3, SwitchClient)
- `window.go` — 6 targets (CreateWindow, CreateArchitectWindow, KillWindowByIndex, FocusWindowByIndex, FocusWindow, ListWindows)
- `command.go` — 2 targets (RunCommandInIndex, SendKeysToIndex)
- `pane.go` — 5 targets (SplitWindowHorizontal, SplitWindowHorizontalWithPercent, ResetWindowPanes, RunCommandInPane, SelectPane)
- `client.go` — 1 target (ListClients)
- `popup.go` — 1 target (DisplayPopup)

Updated test assertions in `tmux_test.go` (2 changes) and `integration_test.go` (3 raw `m.run` targets).

Not changed: `CreateSession` uses `-s name` (literal creation, not target lookup). `validateTmuxName` already blocks `=` in session names.

## Verification

- `make lint` — 0 issues
- `make test` — all unit tests pass
- Commit: `89dd344` on main