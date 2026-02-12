---
id: 6c6368ae-d88b-4551-923b-f9a3417e89fe
author: claude
type: done
created: 2026-02-10T07:22:29.338368Z
---
## Root Cause

Tmux's `-t` flag does prefix matching on bare session names. `has-session -t cortex` incorrectly matches `cortex-meta`, causing `SessionExists("cortex")` to return true when only a `cortex-meta` session exists. This caused architect sessions for projects named "cortex" to reuse the meta session instead of creating a new one. The codebase had already fixed this in `CreateWindow` (with an explicit comment) but the fix was never applied to the other 7 session-targeted functions.

## Resolution

Appended trailing colon (`":"`) to the `-t` argument in all functions that pass bare session names to tmux commands, forcing exact session name resolution:

- `SessionExists` (session.go) — primary bug
- `KillSession` (session.go)
- `AttachSession` (session.go) — both switch-client and attach-session paths
- `SwitchClient` (session.go)
- `ListWindows` (window.go)
- `ListClients` (client.go)
- `DisplayPopup` (popup.go)

Updated `TestSwitchClient` unit test and added `TestIntegrationSessionExistsExactMatch` integration test.

## Files Changed

- `internal/tmux/session.go` — 5 locations fixed
- `internal/tmux/window.go` — 1 location fixed
- `internal/tmux/client.go` — 1 location fixed
- `internal/tmux/popup.go` — 1 location fixed
- `internal/tmux/tmux_test.go` — unit test updated
- `internal/tmux/integration_test.go` — new exact-match test added

## Verification

- `make test` — all unit tests pass
- `make lint` — 0 issues
- Merged to main via fast-forward