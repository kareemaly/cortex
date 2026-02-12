---
id: 6cbde2f8-45de-425c-92a5-66dde732be2c
author: claude
type: review_requested
created: 2026-02-10T14:44:29.887279Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Root Cause

tmux resolves `-t name:` using a cascade: exact match → fnmatch → substring matching. The trailing colon (`name+":"`) only separates session:window — it does NOT disable substring matching for the session part. So `tmux has-session -t cortex:` resolves "cortex" as a substring of "cortex-meta" and returns success, causing `SessionExists("cortex")` → `true` → false "already spawned" detection.

## Fix

Added a `sessionTarget(name)` helper in `internal/tmux/tmux.go` that returns `"=" + name`, leveraging tmux's documented `=` prefix for exact name matching (supported since tmux 2.1, 2015). Applied it consistently across all 22 tmux target constructions in 7 source files:

- `tmux.go` — added `sessionTarget()` helper
- `session.go` — 6 target changes (SessionExists, KillSession, AttachSession x3, SwitchClient)
- `window.go` — 6 target changes (CreateWindow, CreateArchitectWindow, KillWindowByIndex, FocusWindowByIndex, FocusWindow, ListWindows)
- `command.go` — 2 target changes (RunCommandInIndex, SendKeysToIndex)
- `pane.go` — 5 target changes (SplitWindowHorizontal, SplitWindowHorizontalWithPercent, ResetWindowPanes, RunCommandInPane, SelectPane)
- `client.go` — 1 target change (ListClients)
- `popup.go` — 1 target change (DisplayPopup)

Updated test assertions in `tmux_test.go` (2 changes) and `integration_test.go` (3 raw `m.run` targets).

**Not changed**: `CreateSession` uses `-s name` (literal name creation, not target lookup). `validateTmuxName` already blocks `=` in session names.

## Verification

- `make lint` — 0 issues
- `make test` — all unit tests pass (including updated `TestSwitchClient` assertion)
- `TestIntegrationSessionExistsExactMatch` validates the exact-match behavior end-to-end