---
id: 69992ea7-6ad8-4d16-a5a5-46fbf543bda0
author: claude
type: done
created: 2026-02-17T18:34:08.983015Z
---
## Summary

Made the `GET /prompts` listing and `POST /prompts/eject` handlers config-driven so custom ticket types defined in `cortex.yaml` appear in the config tab.

### Changes (single file: `internal/daemon/api/prompts.go`)

**List handler**: Replaced `filepath.WalkDir` with explicit resolver-based enumeration. Prompts are now discovered by:
- Architect: SYSTEM, KICKOFF stages
- Meta: SYSTEM, KICKOFF stages
- Ticket: iterates `cfg.Ticket` keys alphabetically, each with SYSTEM, KICKOFF, APPROVE stages

The resolver handles all fallback logic (custom type → "work" → embedded defaults). Ejection is determined by whether a file exists at `.cortex/prompts/{role}/{type}/{stage}.md`.

**Eject handler**: When the source file doesn't exist on disk (custom types like `ticket/research/SYSTEM.md`), uses the resolver to get fallback content and writes it directly to the destination.

**New helper `resolvePromptByPath()`**: Parses prompt paths into role/type/stage components and dispatches to the appropriate resolver method.

### Verification
- `make build` — compiles
- `make test` — all tests pass
- `make lint` — 0 issues
- Pushed to origin/main (commit 828f790)