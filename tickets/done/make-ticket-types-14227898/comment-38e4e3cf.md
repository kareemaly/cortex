---
id: 38e4e3cf-b32c-4f8c-a25c-ed6a1617e500
author: claude
type: review_requested
created: 2026-02-17T15:56:03.561813Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
        commit: 181dee1
---
## Summary

Makes ticket types fully config-driven by removing all hardcoded `debug`/`research` references.

### Changes (22 files, +76/-206 lines):

1. **`createDoc` available to all ticket agents** — Removed the `if s.session.TicketType == "research"` guard in `tools_ticket.go`. All ticket types can now create docs.

2. **Prompt fallback chain** — Updated `resolver.go` so that when a custom ticket type has no type-specific prompts, it falls back to `work` type prompts: project `{type}` → base `{type}` → project `work` → base `work` → NotFoundError.

3. **Deleted debug/research embedded defaults** — Removed 6 prompt files under `internal/install/defaults/main/prompts/ticket/debug/` and `research/`. Only `work/` remains as the universal fallback.

4. **Hash-based badge colors** — Replaced hardcoded `debugTypeBadgeStyle`/`researchTypeBadgeStyle`/`workTypeBadgeStyle` in both `kanban/styles.go` and `ticket/styles.go` with a hash-based approach using an 8-color ANSI 256 palette. Same type always gets the same color. `work` gets no special styling.

5. **Removed hardcoded type references** — Updated comments, jsonschema descriptions, flag help text across `types.go`, `server.go`, `config.go`, `spawn.go`, `mcp.go`, architect and meta SYSTEM.md prompts, and CLAUDE.md.

6. **Simplified init scaffolding** — `install.go` now only generates `work:` entries for both claude and opencode agents. Users add custom types as needed.

7. **Updated tests** — `embed_test.go` no longer asserts debug/research files exist; `spawn_test.go` uses `"work"` instead of `"research"`; `server_test.go` uses `"custom"` to test generic type plumbing.

### Verification
- `make build` — passes
- `make lint` — 0 issues
- `make test` — all tests pass
- `grep` confirms no remaining hardcoded debug/research ticket type references in Go source