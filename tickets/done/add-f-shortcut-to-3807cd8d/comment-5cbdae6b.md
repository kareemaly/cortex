---
id: 5cbdae6b-6ab5-41bc-aca6-168ff1246d36
author: claude
type: ticket_done
created: 2026-01-26T17:56:47.408508Z
---
## Summary

Added an "f" keyboard shortcut in the kanban TUI that focuses (selects) the tmux window of a ticket's active agent session. This enables quick navigation from the kanban board to a running agent's tmux window.

## Changes Made

### 1. API Endpoint — `internal/daemon/api/tickets.go`
Added `Focus` handler for `POST /tickets/{id}/focus`:
- Looks up the ticket by ID via `store.Get(id)`
- Validates the ticket has an active session with a tmux window; returns 404 with `no_active_session` if not
- Checks tmux manager availability; returns 503 if unavailable
- Loads project config to resolve the tmux session name (same pattern as `Conclude` handler)
- Calls `h.deps.TmuxManager.FocusWindow(tmuxSession, t.Session.TmuxWindow)`
- Returns 200 with `FocusResponse{Success: true, Window: windowName}`

### 2. Route Registration — `internal/daemon/api/server.go`
Registered `r.Post("/{id}/focus", ticketHandlers.Focus)` in the tickets route group alongside other ID-only routes.

### 3. API Response Type — `internal/daemon/api/types.go`
Added `FocusResponse` struct with `Success bool` and `Window string` fields.

### 4. SDK Client Method — `internal/cli/sdk/client.go`
Added `FocusTicket(ticketID string) error` — a fire-and-forget POST following the same pattern as `ApproveSession`.

### 5. TUI Keybinding — `internal/cli/tui/kanban/keys.go`
- Added `KeyFocus Key = "f"` constant (same physical key as `KeyFresh` but used in different context — main kanban vs orphan modal)
- Updated help bar to include `[f]ocus`

### 6. TUI Handler — `internal/cli/tui/kanban/model.go`
- Added `FocusSuccessMsg` and `FocusErrorMsg` message types
- Added "f" key handler: shows "Focusing window..." then calls async `focusTicket`, or shows "No active session" if ticket has no active session
- Added `focusTicket()` async command method
- Added `FocusSuccessMsg`/`FocusErrorMsg` cases in `Update` switch for status feedback

## Key Decisions

- **Reused existing `FocusWindow`**: The tmux layer already had `FocusWindow(session, windowName)` at `internal/tmux/window.go:104-117`, so no tmux-level changes were needed.
- **`KeyFocus` vs `KeyFresh` coexistence**: Both map to "f" but are used in mutually exclusive contexts (main kanban handler vs orphan modal handler), so they coexist without conflict.
- **Fire-and-forget SDK pattern**: `FocusTicket` returns only an error (no response body parsing), matching the `ApproveSession` pattern since callers don't need the response data.

## Files Modified

| File | Lines Changed |
|------|--------------|
| `internal/daemon/api/types.go` | +6 |
| `internal/daemon/api/tickets.go` | +44 |
| `internal/daemon/api/server.go` | +1 |
| `internal/cli/sdk/client.go` | +20 |
| `internal/cli/tui/kanban/keys.go` | +2, -1 |
| `internal/cli/tui/kanban/model.go` | +42 |

## Verification

- `make build` — compiles successfully
- `make lint` — 0 issues
- `make test` — all tests pass