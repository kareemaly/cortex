---
id: e003a54b-3cd4-44fd-80ce-af5d2d097f1e
title: 'Shared TUI Log Viewer: In-Memory Ring Buffer with ! Shortcut'
type: ""
created: 2026-01-27T13:55:15.478061Z
updated: 2026-01-27T14:14:14.793729Z
---
## Summary

Add shared logging infrastructure for all TUI views (kanban, dashboard, ticket detail). An in-memory ring buffer captures logs from SDK client, SSE subscriber, and command handlers. A `!` shortcut opens an overlay log viewer for real-time debugging.

## Design

### Log Buffer (`internal/cli/tui/tuilog/`)

Shared package providing an in-memory ring buffer:

```go
type Level int // Debug, Info, Warn, Error

type Entry struct {
    Time    time.Time
    Level   Level
    Source  string // "sse", "api", "spawn", "tui", etc.
    Message string
}

type Buffer struct {
    // Thread-safe ring buffer, ~1000 entries
}

func (b *Buffer) Log(level Level, source, message string)
func (b *Buffer) Entries() []Entry
func (b *Buffer) ErrorCount() int
func (b *Buffer) WarnCount() int
```

- Thread-safe (TUI and background goroutines write concurrently)
- Fixed capacity ring buffer — oldest entries evicted when full
- No file I/O, purely in-memory

### Log Viewer Overlay

- Toggled with `!` shortcut from any TUI view
- Overlay panel (not a page navigation — preserves underlying view state)
- Newest entries first
- Navigate with `j`/`k`, scroll with `Ctrl+D`/`Ctrl+U`
- Filter by level: `1` = all, `2` = info+, `3` = warn+, `4` = error only
- `!` or `Esc` to dismiss
- Status bar badge shows error/warning count (e.g., `⚠ 2 errors`) when the viewer is closed, so user knows to check

### Integration Points

Log calls to add across TUI infrastructure:

**SSE subscriber:**
- `info` "sse" — "connected to event stream"
- `error` "sse" — "failed to connect: {err}"
- `debug` "sse" — "event received: {type} ticket={id}"
- `warn` "sse" — "connection dropped, no reconnect"

**SDK client / API calls:**
- `error` "api" — "GET /tickets failed: {status} {err}"
- `debug` "api" — "GET /tickets 200 ({duration})"
- `error` "api" — "POST /spawn failed: {err}"

**TUI commands (spawn, focus, move, approve):**
- `info` "spawn" — "spawning session for {ticket}"
- `error` "spawn" — "spawn failed: {err}"
- `info` "focus" — "focusing session {window}"

### Wiring

- Create `Buffer` at TUI entry point (kanban.go, dashboard.go, etc.)
- Pass buffer pointer into the TUI model
- Pass buffer into SDK client or wrap SDK calls with logging
- Each TUI embeds a `LogViewer` bubbletea model that reads from the buffer
- `!` key toggles the overlay in the top-level Update handler

## Files

- `internal/cli/tui/tuilog/buffer.go` — ring buffer implementation
- `internal/cli/tui/tuilog/entry.go` — entry types and levels
- `internal/cli/tui/tuilog/viewer.go` — bubbletea model for the log overlay
- `internal/cli/tui/kanban/model.go` — integrate buffer + viewer
- `internal/cli/tui/dashboard/model.go` — integrate buffer + viewer
- `internal/cli/tui/kanban/model.go` — add log calls to SSE, API, commands
- `internal/cli/tui/dashboard/model.go` — add log calls to SSE, API, commands