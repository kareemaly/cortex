---
id: 4f6e9d2d-5efb-4228-a677-50df2bb4796f
author: claude
type: done
created: 2026-02-07T12:40:36.178885Z
---
## Summary

Implemented persistence of architect session summaries as docs when `concludeSession` is called.

### Change
**`internal/daemon/api/architect.go`** — Added 15 lines to the `Conclude` method:
- After ending the session and before killing the tmux window, creates a doc via `DocsStoreManager.GetStore()`
- Doc title: `"Architect Session — <UTC timestamp>"`
- Category: `sessions`, Tags: `["architect", "session-summary"]`
- Best-effort: failures log a warning but don't block the conclude operation

### Decisions
- Placed doc creation between session end and tmux kill — session is already ended so the summary is saved even if tmux cleanup fails
- Used `DocsStoreManager` already available on `h.deps` — no wiring changes needed
- Used em-dash (—) in title to match existing naming conventions

### Verification
- `make build` — clean
- `make test` — all pass
- `make lint` — 0 issues
- Pre-push hooks passed

### Commit
`40dcbf5` — `feat: persist architect session summary as a doc on conclude`