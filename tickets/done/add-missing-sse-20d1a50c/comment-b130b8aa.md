---
id: b130b8aa-7abf-499f-99f3-172c9786927b
author: claude
type: done
created: 2026-02-15T11:35:05.051939Z
---
## Summary

Implemented missing SSE event emissions and fixed event routing so the TUI can refresh on session lifecycle changes.

### Changes (9 files, +55/-2)

**Core fix — emit 3 missing event types across 6 call sites:**
- `SessionStarted`: emitted in `tickets.go` Spawn() and `architect.go` spawnArchitectSession()
- `SessionEnded`: emitted in `sessions.go` Kill(), `tickets.go` Conclude(), and `architect.go` Conclude()
- `ReviewRequested`: emitted in `tickets.go` RequestReview()

**Event bus improvement:**
- `internal/events/bus.go`: Added `slog.Warn` logging when events are dropped due to full subscriber buffers

**Path normalization fix:**
- `internal/daemon/api/middleware.go`: Applied `filepath.Clean()` to project path in `ProjectRequired` middleware, preventing trailing-slash mismatches between SSE subscriptions and event emissions

**Test fixes:**
- Added `Bus: events.NewBus()` to all test `Dependencies` structs in `tickets_test.go`, `integration_test.go`, `tags_test.go`, and `mcp/tools_test.go`

### Verification
- `make build` — passes
- `make lint` — 0 issues
- `make test` — all unit tests pass
- Pre-push hooks passed on push to main