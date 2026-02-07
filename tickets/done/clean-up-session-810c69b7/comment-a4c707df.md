---
id: a4c707df-05f6-4c31-b7e0-caf55f3b7544
author: claude
type: review_requested
created: 2026-02-07T13:02:51.989369Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
        commit: dd002cb
---
## Summary

Added an explicit `type` field to the `Session` struct to cleanly distinguish architect vs ticket sessions, replacing the previous pattern of checking `ticket_id == "architect"`.

### Changes by file:

1. **`internal/session/session.go`** — Added `SessionType` type with `SessionTypeArchitect`/`SessionTypeTicket` constants. Added `Type` field to `Session` struct. Changed `TicketID` JSON tag to `omitempty`.

2. **`internal/session/store.go`** — `CreateArchitect()` now sets `Type: SessionTypeArchitect` without `TicketID`. `Create()` sets `Type: SessionTypeTicket`. Added backward compat migration in `load()`: old sessions missing `type` are auto-migrated (architect entries get `type=architect` and `ticket_id` cleared; ticket entries get `type=ticket`).

3. **`internal/session/store_test.go`** — Added 3 new tests: `TestCreateSetsTicketType`, `TestCreateArchitectSetsType`, `TestBackwardCompatMigration` (writes old-format JSON and verifies migration).

4. **`internal/types/response.go`** — Added `Type` field to `SessionResponse`, made `TicketID` omitempty.

5. **`internal/types/convert.go`** — Maps `s.Type` to `resp.Type` in `ToSessionResponse()`.

6. **`internal/daemon/api/sessions.go`** — Replaced `shortID == session.ArchitectSessionKey` with `sess.Type == session.SessionTypeArchitect`.

7. **`internal/core/spawn/spawn_test.go`** — Updated mock `Create`/`CreateArchitect` to set the `Type` field.

### NOT changed (intentional):
- `internal/core/spawn/spawn.go` — env var `CORTEX_TICKET_ID=architect` kept for backward compat
- `internal/daemon/api/agent.go` — still checks `req.TicketID == ArchitectSessionKey` (this is the env var value from the HTTP request)
- `internal/daemon/api/architect.go` — conclude response keeps `TicketID: ArchitectSessionKey` for API backward compat

### Verification:
- `make test` — all pass
- `make lint` — 0 issues
- `make build` — clean