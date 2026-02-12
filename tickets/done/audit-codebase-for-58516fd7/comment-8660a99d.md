---
id: 8660a99d-9aac-4629-9559-092acffdefd6
author: claude
type: review_requested
created: 2026-02-08T13:06:09.176135Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## OSS Readiness Audit — Complete

Thorough audit of the Cortex codebase across 11 dimensions. Ran 6 parallel investigation agents covering code quality, public API/naming, error handling, deps/security/config, test coverage, and docs/structure.

### Key Findings (18 items, prioritized):

**P0 — Must Fix (3):**
1. Missing `LICENSE` file (MIT referenced but no file)
2. Daemon binds to `0.0.0.0` instead of `127.0.0.1` — security risk
3. Module path uses personal GitHub handle (`github.com/kareemaly/cortex`)

**P1 — Should Fix (6):**
4. Daemon URL hardcoded in 4 locations instead of centralized config
5. Mixed HTTP error response formats (JSON vs plain text)
6. 6 duplicate type definitions between SDK and API
7. File permissions too permissive (PID, config, backups)
8. SSE event stream silently drops errors
9. Test coverage gaps — SDK client, upgrade, autostart packages have zero tests; tickets.go (964 lines) has no unit tests

**P2 — Nice to Have (9):**
10-18. Error wrapping, 2 logging inconsistencies, missing CODE_OF_CONDUCT, naming polish, misplaced interface, CLI godoc gaps, buried CONFIG_DOCS.md, magic timeout constants, .gitignore gaps

**Positives:** No TODOs/dead code/debug artifacts, no secrets, clean deps (all MIT/Apache), solid architecture, consistent patterns, accurate CLAUDE.md, good test quality where tests exist.

All findings documented with exact file paths and line numbers in ticket comments.