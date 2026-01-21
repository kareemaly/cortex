# Cortex Kanban TUI

Implement the `cortex kanban` command with a Bubbletea-based kanban board.

## Context

The kanban view shows tickets in three columns (backlog, progress, done) with keyboard navigation and actions.

See `DESIGN.md` for:
- Kanban TUI mockup (lines 299-311)
- Navigation keys (line 310)
- CLI command (line 46)

Existing packages:
- `internal/cli/sdk` - client for daemon API
- `internal/cli/tui` - empty, create components here

Dependencies to add:
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - styling

## Requirements

Implement `cortex kanban` command with Bubbletea TUI:

1. **Layout**
   - Three columns: Backlog, Progress, Done
   - Show ticket count per column
   - Ticket cards show title (truncated if needed)
   - Progress tickets show agent status if available

2. **Navigation**
   - `j/k` or arrows: move up/down within column
   - `h/l` or arrows: move between columns
   - `Enter`: view ticket details (could open session view or expand)
   - `s`: spawn session for selected ticket
   - `a`: open architect session
   - `q`: quit

3. **Data Loading**
   - Fetch tickets from daemon API (GET /tickets)
   - Handle daemon not running gracefully
   - Consider periodic refresh or manual refresh key

4. **Styling**
   - Use lipgloss for consistent styling
   - Color code by status (suggestions: backlog=white, progress=yellow, done=green)
   - Highlight selected ticket

## Verification

```bash
make build
make lint
make test

# Manual test
cortexd &  # Start daemon
cortex kanban  # Should show board
```

## Notes

- Reference `~/projects/cortex` (cortex0) for bubbletea patterns
- Start simple - basic navigation and display first
- Actions (spawn, view) can shell out or call SDK
- Handle terminal resize gracefully

## Implementation

### Commits Pushed

- `1a23858` feat: implement kanban TUI with Bubbletea

### Key Files Changed

- `cmd/cortex/commands/kanban.go` - Updated to launch TUI with alt-screen mode
- `internal/cli/tui/kanban/styles.go` - Lipgloss styles (backlog=gray, progress=yellow, done=green)
- `internal/cli/tui/kanban/column.go` - Column component with cursor navigation
- `internal/cli/tui/kanban/keys.go` - Key bindings (vim-style h/j/k/l)
- `internal/cli/tui/kanban/model.go` - Main Bubbletea model with async loading
- `go.mod`, `go.sum` - Added bubbletea and lipgloss dependencies

### Important Decisions

- Used vim-style keybindings as primary (h/j/k/l) with arrow key support
- Active sessions shown with bullet indicator (●) next to ticket title
- Architect mode (`a` key) returns placeholder error - to be implemented separately
- Status messages auto-clear after 3 seconds
- Columns have minimum width of 20 chars for readability

### Scope Changes

- Ticket details view (`Enter` key) not implemented - marked for future enhancement
- Agent status display in Progress column deferred - shows active session indicator instead
