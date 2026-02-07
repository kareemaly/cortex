---
id: 01a18590-f720-4282-9acd-75fa390c7bc3
author: claude
type: ticket_done
created: 2026-01-27T12:15:38.976154Z
---
## Daemon Dashboard TUI - Implementation Report

### Summary
Implemented two new CLI commands that together form the Cortex command center: `cortex dashboard` (a global bubbletea TUI showing all registered projects with active sessions) and `cortex start` (a tmux orchestrator creating a CortexDaemon session with dashboard + daemon panes).

### Files Created (5)

1. **`cmd/cortex/commands/dashboard.go`** — `cortex dashboard` cobra command. Global command (no project context required). Creates an `sdk.DefaultClient("")` and launches the dashboard TUI with `tea.WithAltScreen()`. Follows the pattern established by `kanban.go`.

2. **`cmd/cortex/commands/start.go`** — `cortex start` cobra command. Creates/attaches to a `CortexDaemon` tmux session. If the session exists, attaches directly. If not, creates the session, splits horizontally (30% right pane for `cortexd`, 70% left for `cortex dashboard`), runs both commands, focuses the dashboard pane, and attaches. Follows the pattern from `architect.go`.

3. **`internal/cli/tui/dashboard/model.go`** (900 lines) — Main bubbletea model implementing:
   - **Data model**: `projectData` structs holding project info, expansion state, loaded tickets, and architect state. Flattened `[]row` tree for cursor navigation with `rowProject` and `rowSession` kinds.
   - **Initial load**: `ListProjects()` via global client. Projects start collapsed with counts from `ProjectResponse.Counts`.
   - **Expand/collapse**: On expand, creates per-project `sdk.DefaultClient(projectPath)` to fetch `ListAllTickets("")` and `GetArchitect()`. On collapse, cancels SSE and clears data.
   - **SSE**: Per-project SSE subscription on expand, cancel on collapse. Any event triggers reload of that project's data. Follows kanban's `subscribeEvents`/`waitForEvent`/`EventMsg` pattern.
   - **Duration tick**: 30-second `tea.Tick` for re-rendering duration display.
   - **Key bindings**: j/k/up/down (navigate), enter/l (expand), h (collapse), f (focus session), s (spawn architect), r (refresh all), gg/G (jump first/last), ctrl+u/ctrl+d (half-page scroll), q/ctrl+c (quit).
   - **View**: Header bar, scrollable tree with project rows (expand indicator + title + counts) and indented session rows (agent status icon + name + status badge + duration), status bar, help bar.

4. **`internal/cli/tui/dashboard/keys.go`** — Key constants and `isKey` helper function, following the kanban pattern.

5. **`internal/cli/tui/dashboard/styles.go`** — Lipgloss styles matching the existing kanban color scheme (selected = white on blue, progress = yellow/orange, review = blue, etc.).

### Files Modified (1)

6. **`internal/tmux/pane.go`** — Added `SplitWindowHorizontalWithPercent(session, windowIndex, percent, workingDir)` method alongside existing `SplitWindowHorizontal`. The existing method hardcodes `-p 70`; the new method accepts a custom percent parameter for the new right pane.

### Key Decisions

- **Pure client architecture**: Dashboard uses the same HTTP API as CLI/MCP/TUI. No daemon changes needed. Dashboard can crash/restart independently.
- **Per-project SSE subscriptions**: Only expanded projects subscribe to SSE, avoiding unnecessary connections. Subscriptions are cancelled on collapse.
- **Flattened tree model**: Projects and sessions are flattened into a `[]row` slice for simple cursor-based navigation, rebuilt on expand/collapse.
- **Reused kanban patterns**: Same key binding style, lipgloss color scheme, SSE subscription pattern, and async command pattern as the existing kanban TUI.
- **No daemon modifications**: All existing API endpoints (`GET /projects`, `GET /tickets`, `GET /architect`, `POST /architect/spawn`, `POST /tickets/{id}/focus`, `GET /events`) are sufficient.

### Verification
- `make build` — passes
- `make lint` — 0 issues
- `make test` — all tests pass
- Merged to main via fast-forward