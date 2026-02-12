---
id: 062db32c-533e-4149-8ec2-25763cd0d411
author: claude
type: done
created: 2026-02-10T09:52:08.025005Z
---
## Summary

Injected ticket comments into the agent's KICKOFF prompt so agents have full context at session start. This was needed because `readTicket` was removed in a prior optimization (ticket 06a6c1c8), leaving agents unable to see prior comments — especially critical for resume/fresh mode spawns where comments contain blocker notes, decisions, and investigation findings.

### Changes (7 files, 252 insertions)

1. **`internal/prompt/template.go`** — Added `Comments string` field to `TicketVars` struct
2. **`internal/core/spawn/spawn.go`** — Added `formatTicketComments()` helper that formats comments as `### [type] — YYYY-MM-DD HH:MM UTC` + content; wired into `buildTicketAgentPrompt()` to populate `vars.Comments`
3. **4 KICKOFF templates** (work, debug, research, chore) — Added `{{if .Comments}}` conditional block that renders a "## Comments" section only when comments exist
4. **`internal/core/spawn/spawn_test.go`** — Added 3 tests: end-to-end with comments, no-comments case, and formatter unit test

### Verification
- `make build` — passes
- `make test` — all tests pass (including 3 new)
- `make lint` — 0 issues
- Pre-push hooks passed
- Merged to main and pushed