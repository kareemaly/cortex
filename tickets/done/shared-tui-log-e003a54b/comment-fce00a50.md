---
id: fce00a50-2413-4287-81bc-bfd29515a478
author: claude
type: ticket_done
created: 2026-01-27T14:14:14.791789Z
---
## Summary

Implemented a shared TUI log viewer infrastructure for the kanban and dashboard views. The feature provides an in-memory ring buffer for capturing operational logs and a full-screen overlay viewer toggled with the `!` key.

## Changes Made

### New Files (4)

- **`internal/cli/tui/tuilog/entry.go`** — `Level` type (Debug/Info/Warn/Error) with `String()` and `ShortString()` methods, and `Entry` struct (Time, Level, Source, Message).

- **`internal/cli/tui/tuilog/buffer.go`** — Thread-safe ring buffer with 1000-entry default capacity. Uses fixed-size `[]Entry` slice with modular index arithmetic and `sync.Mutex`. Tracks error/warn counts in O(1) by incrementing on write and decrementing when overwriting old entries. Provides convenience methods: `Debug/Info/Warn/Error` and `Debugf/Infof/Warnf/Errorf`.

- **`internal/cli/tui/tuilog/buffer_test.go`** — 7 test cases covering: basic log + newest-first ordering, ring wrap behavior, error/warn count tracking, counts through overwrites, empty buffer, formatted methods, and level string methods.

- **`internal/cli/tui/tuilog/viewer.go`** — Bubbletea sub-model rendering a full-screen overlay. Features: j/k scroll, ctrl+d/u page scroll, 1-4 level filters (all/info+/warn+/error), !/esc dismiss. Styled with dark red title bar, colored level indicators (red=error, orange=warn, green=info, gray=debug), blue source, gray timestamps.

### Modified Files (8)

- **`internal/cli/tui/kanban/keys.go`** — Added `KeyExclaim` constant, updated help text with `[!] logs`.
- **`internal/cli/tui/kanban/styles.go`** — Added `warnBadgeStyle` (foreground Color("214")).
- **`internal/cli/tui/kanban/model.go`** — Added `logBuf`, `logViewer`, `showLogViewer` fields. Changed `New()` to accept `*tuilog.Buffer`. Added DismissLogViewerMsg handler and viewer delegation at top of Update. Added log calls to 9 message handlers (TicketsLoaded, TicketsError, SessionSpawned, SessionError, OrphanedSession, FocusSuccess, FocusError, sseConnected, EventMsg). Added `!` toggle in handleKeyMsg. Added log viewer overlay in View. Added `logBadge()` helper rendering E:/W: counts in status bar.
- **`cmd/cortex/commands/kanban.go`** — Creates `logBuf` and passes to `kanban.New()`.
- **`internal/cli/tui/dashboard/keys.go`** — Added `KeyExclaim`, updated help text.
- **`internal/cli/tui/dashboard/styles.go`** — Added `warnBadgeStyle`.
- **`internal/cli/tui/dashboard/model.go`** — Same pattern as kanban: added fields, changed constructor, added viewer handling, log calls to 8 message handlers (ProjectsLoaded, ProjectsError, ProjectDetailLoaded, SSEConnected, SSEEvent, SpawnArchitect, FocusSuccess, FocusError), `!` toggle, overlay view, `logBadge()`.
- **`cmd/cortex/commands/dashboard.go`** — Creates `logBuf` and passes to `dashboard.New()`.

## Key Decisions

- **Ring buffer over unbounded list**: Fixed 1000-entry capacity prevents memory growth in long-running TUI sessions. Old entries are silently evicted.
- **O(1) count tracking**: Error/warn counts are maintained incrementally rather than recomputed on each access, keeping the status bar badge efficient.
- **Viewer reads fresh on each frame**: `View()` calls `buf.Entries()` every render rather than caching, ensuring the overlay always shows latest logs without additional message plumbing.
- **Used `any` instead of `interface{}`**: Addressed linter modernize warnings for Go 1.24 compatibility.
- **No ticket detail view integration**: As specified in the plan, the ticket detail view doesn't get a log viewer since it has no background operations worth logging.

## Verification

- `make build` — compiles cleanly
- `make lint` — 0 issues
- `make test` — all tests pass (7 new tuilog tests + all existing tests)