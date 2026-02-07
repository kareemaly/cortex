---
id: 3807cd8d-d16a-45b8-b29b-691dfa37f225
title: Add "f" Shortcut to Focus Ticket's Tmux Window in TUI
type: ""
created: 2026-01-26T17:14:22.799893Z
updated: 2026-01-26T17:56:47.410018Z
---
## Context

This project is in early development. Breaking changes are fine. Do not accumulate tech debt — write clean, direct code without backwards-compatibility shims or unnecessary abstractions.

## Problem

When a ticket has an active agent session running in a tmux window, there's no quick way to jump to that window from the kanban TUI. Users have to manually find and switch to the tmux window.

## Solution

Add an "f" keyboard shortcut in the kanban TUI that, when pressed on a ticket with an active session, calls the daemon API to focus (select) the ticket's tmux window.

## Scope

### 1. Daemon API Endpoint

Add `POST /tickets/{id}/focus` endpoint:

- Look up the ticket's active session to get `tmux_window` name
- Call tmux to select that window
- Return 404 if no active session or window not found
- Return 200 on success

### 2. SDK Client Method

Add `FocusTicket(ticketID string) error` to the SDK client.

### 3. TUI Keybinding

In the kanban TUI:

- Add "f" keybinding when a ticket is selected
- Call SDK `FocusTicket()` with the selected ticket ID
- Show a status message on success ("Focused on window: ...") or error ("No active session")
- Add "f focus" to the help/footer bar

### 4. Tmux Focus Logic

In `internal/tmux/`:

- Add a `FocusWindow(sessionName, windowName string) error` function
- Uses `tmux select-window -t {session}:{window}` to switch

## Key Files

| File | Change |
|------|--------|
| `internal/tmux/window.go` | Add `FocusWindow()` function |
| `internal/daemon/api/tickets.go` | Add focus endpoint handler |
| `internal/daemon/api/routes.go` | Register `POST /tickets/{id}/focus` |
| `internal/cli/sdk/client.go` | Add `FocusTicket()` method |
| TUI keybinding file | Add "f" handler, call SDK, show status |

## Acceptance Criteria

- [ ] `POST /tickets/{id}/focus` endpoint exists and switches tmux window
- [ ] SDK client has `FocusTicket()` method
- [ ] Pressing "f" on a ticket in the TUI focuses its tmux window
- [ ] Appropriate error shown if ticket has no active session
- [ ] Help bar shows "f focus" hint