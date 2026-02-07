---
id: 9b3e4847-c68c-48c1-a98c-3af08937c3a4
author: claude
type: ticket_done
created: 2026-01-27T14:44:43.151584Z
---
## Summary

Added a `ga` (go architect) two-key shortcut to the ticket detail TUI that focuses the architect window (window 0) of the project's tmux session. This follows the same `pendingG` pattern established by the kanban's `gd` shortcut.

## Changes Made

### 1. `internal/daemon/api/architect.go`
- Added `Focus` method to `ArchitectHandlers`
- Loads project config to get tmux session name (same pattern as `GetState`/`Spawn`)
- Calls `FocusWindowByIndex(sessionName, 0)` to select window 0 (architect)
- Calls `SwitchClient(sessionName)` for cross-session switching (non-fatal on failure)
- Returns `FocusResponse{Success: true, Window: "architect"}`

### 2. `internal/daemon/api/server.go`
- Added `r.Post("/focus", architectHandlers.Focus)` route inside the `/architect` group

### 3. `internal/cli/sdk/client.go`
- Added `FocusArchitect()` method — `POST /architect/focus` with project header via `doRequest`
- Follows same pattern as `FocusTicket()` and `FocusDaemonDashboard()`

### 4. `internal/cli/tui/ticket/model.go`
- Added `ga` key handler: `if m.pendingG && isKey(msg, KeyApprove)` — clears pendingG and calls `focusArchitect()`
- Guarded approve handler with `!m.pendingG` to prevent `a` from triggering approve during `ga` sequence
- Added fire-and-forget `focusArchitect()` tea.Cmd (on success user lands on architect window; on failure they stay in TUI)

### 5. `internal/cli/tui/ticket/keys.go`
- Updated all help text variants from `[j/k/gg/G]` to `[j/k/gg/G/ga]`

## Key Decisions

- **Guard on approve handler**: Since `a` is `KeyApprove`, pressing `ga` would trigger approve on the second keystroke. Added `!m.pendingG` guard to prevent this, ensuring `a` after `g` is always interpreted as `ga`.
- **Fire-and-forget pattern**: The `focusArchitect()` command ignores errors — on success the tmux focus switches to the architect window; on failure the user remains in the TUI with no disruption.
- **`SwitchClient` non-fatal**: Matches the existing `DaemonFocusHandler` pattern where `SwitchClient` failure is logged but doesn't block the response.

## Verification

- `make build` — compiles successfully
- `make lint` — 0 issues
- `make test` — all unit tests pass