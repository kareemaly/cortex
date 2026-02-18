---
id: f93247fc-86d3-4746-953d-fc887a1793ab
author: claude
type: done
created: 2026-02-12T13:04:30.591768Z
---
## Summary

Implemented sorting of dashboard sessions by `SessionStartedAt` (most recent first).

### Changes (3 files, commit `5fb1a1b`)

1. **`internal/types/response.go`** — Added `SessionStartedAt *time.Time` field to `TicketSummary` struct
2. **`internal/types/convert.go`** — Populated `SessionStartedAt` from `sess.StartedAt` in `ToTicketSummary`
3. **`internal/cli/tui/dashboard/model.go`** — Three enhancements:
   - Session rows sorted by `SessionStartedAt` descending (most recent first)
   - Duration display uses session start time instead of ticket update time
   - Active projects sorted by newest session time via `newestSessionTime` helper

### Verification
- `make build` — clean
- `make test` — all tests pass
- `make lint` — 0 issues
- Pre-push hooks passed