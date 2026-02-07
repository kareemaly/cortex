---
id: 390e6d2c-c896-414c-a9be-58c08446597c
author: claude
type: done
created: 2026-02-07T13:11:09.640998Z
---
## Summary

Added an explicit `type` field to the `Session` struct to cleanly distinguish architect vs ticket sessions, replacing the fragile pattern of checking `ticket_id == "architect"`.

### Changes (7 files, 125 insertions, 5 deletions):

**Core model:**
- `internal/session/session.go` — Added `SessionType` type (`"architect"` / `"ticket"`), `Type` field on `Session`, made `TicketID` `omitempty`
- `internal/session/store.go` — `CreateArchitect()` sets `Type: SessionTypeArchitect` with no `TicketID`; `Create()` sets `Type: SessionTypeTicket`; backward compat migration in `load()` auto-fixes old sessions.json

**API layer:**
- `internal/types/response.go` — Added `Type` field to `SessionResponse`, made `TicketID` omitempty
- `internal/types/convert.go` — Maps `s.Type` → `resp.Type` in `ToSessionResponse()`
- `internal/daemon/api/sessions.go` — Session list uses `sess.Type == SessionTypeArchitect` instead of key comparison

**Tests:**
- `internal/session/store_test.go` — 3 new tests: type assignment for Create/CreateArchitect, backward compat migration
- `internal/core/spawn/spawn_test.go` — Updated mocks to set `Type` field

### Intentionally unchanged:
- `spawn.go` env var `CORTEX_TICKET_ID=architect` kept for lifecycle hook backward compat
- `agent.go` still checks `req.TicketID == ArchitectSessionKey` (env var from HTTP request)
- `architect.go` conclude response keeps `TicketID: ArchitectSessionKey` for API compat

### No documentation changes needed — internal data model refactor only.

All tests pass, lint clean, build clean, pushed to origin.