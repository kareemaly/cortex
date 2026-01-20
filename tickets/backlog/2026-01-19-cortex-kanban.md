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
