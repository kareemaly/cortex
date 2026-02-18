---
id: a306561d-cb78-4eca-9348-3af9c9f08feb
author: claude
type: done
created: 2026-02-14T11:33:58.814414Z
---
## Summary

Wired existing ticket `references` field through to kickoff prompt templates so agents see referenced tickets/docs at spawn time.

### Files Changed

- **`internal/prompt/template.go`** — Added `References string` field to `TicketVars` struct
- **`internal/core/spawn/spawn.go`** — Added `formatTicketReferences()` helper and populated `vars.References` in `buildTicketAgentPrompt`
- **`internal/install/defaults/main/prompts/ticket/{work,research,debug,chore}/KICKOFF.md`** — Added conditional `## References` section after ticket body
- **`internal/prompt/template_test.go`** — Added two tests for references rendering (present and omitted when empty)

### Commit

`88b7904` — `feat: render ticket references in kickoff prompts`

Pushed to origin/main. All tests pass, lint clean, pre-push hooks passed.