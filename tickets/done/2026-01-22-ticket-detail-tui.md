# Ticket Detail TUI

Create a rich TUI view for `cortex ticket <id>` command.

## Context

This is a fresh project with no users. No backward compatibility needed. Breaking changes are fine. Do not accumulate tech debt.

## Current State

`cortex ticket <id>` currently shows plain text output. Convert to interactive TUI.

## TUI Display

Show ticket information in a scrollable view:

### Header
- Ticket ID and status (with color)
- Title

### Body
- Full ticket body/description

### Session Info (if exists)
- Session ID, agent, tmux window
- Started at, ended at
- State indicator (active/orphaned/ended)

### Comments
- List of comments with:
  - Type (color-coded: decision=blue, scope_change=yellow, blocker=red, etc.)
  - Content
  - Timestamp

## Interactions

- `q` - quit
- `r` - refresh
- Arrow keys / j/k - scroll

## Fallback

Keep `--json` flag for non-interactive output (skip TUI, print JSON).

## Files to Change

- `cmd/cortex/commands/ticket_show.go` - launch TUI instead of text
- `internal/cli/tui/ticket/` (new) - ticket detail TUI components

## Verification

```bash
make lint
make test
make build
make test-integration
```

## Implementation

### Commits Pushed

- `e1d8276` feat: add interactive TUI for ticket detail view with scrollable viewport

### Key Files Changed

**New files:**
- `internal/cli/tui/ticket/keys.go` - Key bindings (q, j/k, pgup/pgdown, home/end, r)
- `internal/cli/tui/ticket/styles.go` - Styling with status colors and comment type colors
- `internal/cli/tui/ticket/model.go` - Main Bubbletea model with viewport for scrollable content

**Modified files:**
- `cmd/cortex/commands/ticket_show.go` - Launches TUI in TTY mode, falls back to plain text when piped
- `go.mod` / `go.sum` - Added `github.com/charmbracelet/bubbles` dependency

### Important Decisions

- Used `charmbracelet/bubbles` viewport component for scrolling (new dependency)
- TTY detection via `mattn/go-isatty` (already an indirect dependency)
- Plain text fallback for non-TTY (piped output) preserves scriptability
- Reused status colors from kanban TUI for consistency
- Comment types color-coded: decision (blue), blocker (red), question (cyan), etc.

### Scope Changes

None - implemented as specified.
