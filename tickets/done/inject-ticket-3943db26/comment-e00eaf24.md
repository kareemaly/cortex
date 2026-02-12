---
id: e00eaf24-c54a-4393-a547-29db35ca2edf
author: claude
type: review_requested
created: 2026-02-10T09:50:23.274114Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/d31ff1fc-53fe-41da-a715-165c558124f7
        commit: 1e73e75
---
## Summary

Injects ticket comments into the agent's KICKOFF prompt so agents have full context at session start. This is needed because `readTicket` was removed in a prior optimization, leaving agents unable to see prior comments (blockers, decisions, investigation findings).

### Changes

1. **`internal/prompt/template.go`** — Added `Comments string` field to `TicketVars`

2. **`internal/core/spawn/spawn.go`** — Added `formatTicketComments()` helper that formats comments as markdown (`### [type] — timestamp` + content), and wired it into `buildTicketAgentPrompt()` to populate `vars.Comments`

3. **KICKOFF templates** (work, debug, research, chore) — Added `{{if .Comments}}` conditional block that renders the comments section only when comments exist

4. **`internal/core/spawn/spawn_test.go`** — Added 3 tests:
   - `TestSpawn_TicketAgent_WithComments` — end-to-end: spawns with comments, reads generated prompt, verifies comment types/content/timestamps appear
   - `TestSpawn_TicketAgent_NoComments` — verifies no "## Comments" section when ticket has no comments
   - `TestFormatTicketComments` — unit test for the formatter (nil, empty, single comment cases)

### Verification
- `make build` — passes
- `make test` — all tests pass (including 3 new)
- `make lint` — 0 issues