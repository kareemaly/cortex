---
id: 5bddc6a6-efbc-446d-99bb-0a282a15c2ea
author: claude
type: review_requested
created: 2026-02-14T11:33:21.486527Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
        commit: 88b7904
---
## Summary

Wired the existing `references` field from ticket metadata through to kickoff prompt templates so agents can see referenced tickets/docs at spawn time.

### Changes

1. **`internal/prompt/template.go`** — Added `References string` field to `TicketVars` struct (pre-formatted, same pattern as `Comments`)

2. **`internal/core/spawn/spawn.go`** — Added `formatTicketReferences` helper that joins references as a bulleted markdown list, and populated `vars.References` in `buildTicketAgentPrompt`

3. **4 KICKOFF templates** (`work`, `research`, `debug`, `chore`) — Added conditional `## References` section after the ticket body, using the same `{{if .Field}}` pattern as existing sections

4. **`internal/prompt/template_test.go`** — Added two tests: one verifying references render correctly, one verifying the section is omitted when empty

### Verification

- `make test` — all unit tests pass
- `make lint` — 0 issues