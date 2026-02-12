---
id: 5f583426-efe7-45af-9543-03752fe0021f
author: claude
type: review_requested
created: 2026-02-10T07:18:45.424347Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/64ace102-98da-46db-a999-998bcf360169
        commit: 43778b0
---
## Root Cause

Tmux's `-t` flag does prefix matching on bare session names. `has-session -t cortex` matches `cortex-meta`, causing `SessionExists("cortex")` to return true when only `cortex-meta` exists. The codebase already fixed this in `CreateWindow` (with an explicit comment explaining the behavior) but never applied the same fix to other session-targeted commands.

## Fix

Appended trailing colon (`":"`) to the `-t` argument in all 7 functions that pass bare session names to tmux commands:

| Function | File | Command |
|---|---|---|
| `SessionExists` | session.go:17 | `has-session -t name:` |
| `KillSession` | session.go:63 | `kill-session -t name:` |
| `AttachSession` | session.go:88,90 | `switch-client/attach-session -t name:` |
| `SwitchClient` | session.go:96 | `switch-client -t session:` |
| `ListWindows` | window.go:157 | `list-windows -t session:` |
| `ListClients` | client.go:30 | `list-clients -t session:` |
| `DisplayPopup` | popup.go:12 | `display-popup -t session:` |

The trailing colon forces tmux to resolve the target as an exact session name (documented in tmux manpage as session target syntax `session:`).

## Tests

- Updated `TestSwitchClient` unit test to expect the trailing colon
- Added `TestIntegrationSessionExistsExactMatch` that creates `cortex-test-{ts}-meta` and verifies the prefix `cortex-test-{ts}` does NOT match
- `make test` — all unit tests pass
- `make lint` — 0 issues