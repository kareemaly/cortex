---
id: 1683e0e7-49a0-446f-8317-7a7a0e8103c4
author: claude
type: review_requested
created: 2026-02-14T11:52:27.283072Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Summary

Removed the `chore` ticket type entirely from the codebase. It was nearly identical to `work` (same KICKOFF template, near-identical APPROVE, trivial SYSTEM prompt difference) and didn't justify a separate type.

### Changes made:

1. **Deleted chore prompt templates** — removed `internal/install/defaults/main/prompts/ticket/chore/` directory (SYSTEM.md, KICKOFF.md, APPROVE.md)

2. **Go source files** — removed all `chore` references from:
   - `cmd/cortexd/commands/mcp.go` — flag description
   - `internal/core/spawn/spawn.go` — comment on ResumeRequest.TicketType
   - `internal/core/spawn/config.go` — comment on MCPConfigParams.TicketType
   - `internal/daemon/mcp/server.go` — comment on Config.TicketType
   - `internal/daemon/mcp/types.go` — Session.TicketType comment, CreateTicketInput.Type jsonschema, ReadPromptInput.TicketType jsonschema, UpdatePromptInput.TicketType jsonschema
   - `internal/install/install.go` — removed chore blocks from both opencode and claude config templates
   - `internal/install/embed_test.go` — removed 3 chore entries from expectedFiles

3. **TUI styles** — removed `choreTypeBadgeStyle` and `case "chore"` from:
   - `internal/cli/tui/kanban/styles.go` — badge style var, typeBadgeColorCode(), typeBadgeStyle()
   - `internal/cli/tui/ticket/styles.go` — badge style var, typeBadgeStyle()

4. **Documentation & prompts** — removed chore from:
   - `README.md` — ticket config example
   - `internal/install/defaults/main/prompts/architect/SYSTEM.md` — ticket types list
   - `internal/install/defaults/main/prompts/meta/SYSTEM.md` — ticket types list

5. **Project config** — removed chore block from `.cortex/cortex.yaml`

6. **Migrated 7 existing chore tickets to work** — changed `type: chore` → `type: work` in frontmatter of all done tickets

### Verification

- `make build` — compiles successfully
- `make test` — all unit tests pass
- `make lint` — 0 issues
- `grep -r chore --include="*.go"` — no remaining Go references