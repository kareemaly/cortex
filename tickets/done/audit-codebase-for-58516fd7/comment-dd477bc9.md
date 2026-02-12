---
id: dd477bc9-619d-47ca-88ed-e54bab6579e9
author: claude
type: done
created: 2026-02-08T13:07:18.425261Z
---
## OSS Readiness Audit — Complete

### Scope
Audited ~31K lines of Go across 30 packages, covering: code quality, public API/naming, error handling, dependencies/security/config, test coverage, and documentation/structure.

### Key Findings (18 items)

**P0 — Must Fix (3):**
1. Missing `LICENSE` file (MIT referenced in README but no file exists)
2. Daemon binds to `0.0.0.0` instead of `127.0.0.1` — security risk (`internal/daemon/api/server.go:113`)
3. Go module path uses personal GitHub handle (`github.com/kareemaly/cortex`) — needs org migration

**P1 — Should Fix (6):**
4. Daemon URL hardcoded in 4 locations instead of centralized config
5. Mixed HTTP error response formats — some handlers use `http.Error()` (plain text) instead of `writeError()` (JSON)
6. 6 duplicate type definitions between `cli/sdk/client.go` and `daemon/api/types.go`
7. File permissions too permissive — PID file, settings.yaml, binary backups all use 0644/0755 instead of 0600/0700
8. SSE event stream silently drops JSON marshal and write errors (`events.go:47-49`)
9. Test coverage gaps — `cli/sdk/client.go` (critical HTTP client), `upgrade/`, `autostart/` have zero tests; `tickets.go` (964 lines) has no unit tests

**P2 — Nice to Have (9):**
10. ~20 instances of `fmt.Errorf` without `%w` wrapping
11. 2 `log.Printf` calls in otherwise-consistent `slog` codebase (`tickets.go:818,832`)
12. Missing `CODE_OF_CONDUCT.md` and `.github/` templates
13. `DocsStoreManager` naming stutters; `CreateComment()` breaks `New*()` convention
14. `TmuxChecker` interface misplaced in `types` package
15. CLI commands 30-40% godoc coverage (core packages are well-documented)
16. `CONFIG_DOCS.md` buried in `internal/install/defaults/claude-code/`
17. Magic timeout values scattered across 4+ files
18. `.gitignore` missing `.env*`, `*.pid`, `.cortex/logs/` patterns

### Positives
- Zero TODOs, dead code, debug artifacts, or hardcoded secrets
- Clean dependency set (12 direct deps, all MIT/Apache 2.0)
- Well-designed custom error types with `Is*()` checkers across 8 packages
- Standard Go project layout, clean `cortex`/`cortexd` separation
- Consistent `New*()` constructors (36 instances), proper mutex safety (26 locks all deferred)
- CLAUDE.md verified 100% accurate against codebase
- Good test quality where tests exist (table-driven, subtests, proper cleanup)

### Recommended Next Steps
1. **Create tickets for P0 items** — LICENSE file, localhost binding, module path decision
2. **Batch P1 items into 2-3 work tickets** — security hardening (permissions + binding), type consolidation, test coverage sprint
3. **Defer P2 items** — address as part of normal development velocity
4. **Consider a test coverage sprint** targeting sdk/client.go, upgrade/, and tickets.go before release