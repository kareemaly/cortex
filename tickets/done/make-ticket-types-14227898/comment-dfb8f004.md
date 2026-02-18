---
id: dfb8f004-a4ad-4470-8e52-03c0184ff206
author: claude
type: done
created: 2026-02-17T16:18:49.290091Z
---
## Summary

Made ticket types fully config-driven by removing all hardcoded `debug`/`research`/`chore` references.

### Code changes (commit `181dee1`, 22 files, +76/-206):

1. **`createDoc` for all ticket agents** — Removed `research`-only guard in `tools_ticket.go`
2. **Prompt fallback chain** — `resolver.go` now falls back to `work` prompts when a custom type has no type-specific prompts (chain: project `{type}` → base `{type}` → project `work` → base `work`)
3. **Deleted debug/research embedded defaults** — Removed 6 prompt files; only `work/` remains under `internal/install/defaults/main/prompts/ticket/`
4. **Hash-based badge colors** — Replaced hardcoded styles in `kanban/styles.go` and `ticket/styles.go` with 8-color ANSI 256 palette, deterministic by type name
5. **Removed hardcoded type references** — Updated comments, jsonschema descriptions, and flag help text across `types.go`, `server.go`, `config.go`, `spawn.go`, `mcp.go`, architect/meta SYSTEM.md prompts, and CLAUDE.md
6. **Simplified init scaffolding** — `install.go` only generates `work:` type for both claude and opencode
7. **Updated tests** — `embed_test.go`, `spawn_test.go`, `server_test.go`

### Config cleanup (commit `ee285ea`):

- Cleaned `~/.cortex/defaults/main/prompts/ticket/` — removed `debug/`, `research/`, `chore/` directories
- Cleaned all 13 registered projects' `cortex.yaml` — removed `debug`/`research`/`chore` entries, keeping only `work` (except lithoapp which retains `research` with custom `--dangerously-skip-permissions` args)

### Verification:
- `make build` ✓, `make lint` (0 issues) ✓, `make test` (all pass) ✓
- Pre-push hooks passed ✓
- Pushed to origin/main