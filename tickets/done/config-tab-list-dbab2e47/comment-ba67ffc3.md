---
id: ba67ffc3-79f0-42cd-8282-8b485d01d153
author: claude
type: review_requested
created: 2026-02-17T16:36:34.046501Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
        commit: 828f790
---
## Changes

Single file modified: `internal/daemon/api/prompts.go`

### List handler (rewritten)
Replaced `filepath.WalkDir` approach with explicit enumeration using the prompt resolver:
- **Architect prompts**: Enumerates SYSTEM, KICKOFF stages via `resolver.ResolveArchitectPromptWithPath()`
- **Meta prompts**: Enumerates SYSTEM, KICKOFF stages via `resolver.ResolveMetaPromptWithPath()`
- **Ticket prompts**: Iterates `cfg.Ticket` keys sorted alphabetically, enumerates SYSTEM, KICKOFF, APPROVE stages per type via `resolver.ResolveTicketPromptWithPath()`

Ejection is determined by whether a project-specific file exists at `.cortex/prompts/{role}/{type}/{stage}.md`. The resolver handles all fallback logic (custom type → "work" → embedded defaults).

### Eject handler (updated)
When the direct source file doesn't exist on disk (custom types like `ticket/research/SYSTEM.md`), falls back to the resolver to get content and writes it directly via `os.WriteFile` instead of `copyPromptFile`.

### New helper: `resolvePromptByPath()`
Parses a prompt path (e.g., `ticket/research/SYSTEM.md`) into role/type/stage components and calls the appropriate resolver method.

### Verification
- `make build` — compiles successfully
- `make test` — all tests pass
- `make lint` — 0 issues