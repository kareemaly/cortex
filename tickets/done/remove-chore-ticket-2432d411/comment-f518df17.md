---
id: f518df17-99d2-4b61-8217-7ed373d0d885
author: claude
type: done
created: 2026-02-14T11:55:16.474781Z
---
## Summary

Removed the `chore` ticket type entirely from the codebase. It was nearly identical to `work` (same KICKOFF template, near-identical APPROVE, trivial SYSTEM prompt difference) and didn't justify a separate type.

## Changes

- **Deleted** `internal/install/defaults/main/prompts/ticket/chore/` (3 template files)
- **Updated 11 Go source files** to remove chore from comments, jsonschema descriptions, flag descriptions, config templates, and test expectations
- **Updated 2 TUI style files** to remove choreTypeBadgeStyle and case "chore" branches
- **Updated 3 documentation/prompt files** (README.md, architect SYSTEM.md, meta SYSTEM.md)
- **Updated project config** `.cortex/cortex.yaml` to remove chore block
- **Migrated 7 existing chore tickets** to `type: work`

## Verification

- `make build` — compiles
- `make test` — all tests pass
- `make lint` — 0 issues
- Pre-push hooks passed

## Commit

`53f01b1` — `refactor: remove chore ticket type entirely`