---
id: 301c3fea-dc5f-417e-8ef1-6b3bde7ff927
author: claude
type: comment
created: 2026-02-08T13:05:35.748866Z
---
## P2 - Nice to Have (Polish)

### 10. Error wrapping inconsistency
~20 instances of `fmt.Errorf("...", val)` without `%w` wrapping, which breaks `errors.Is()`/`errors.As()` chains. Most are in `internal/upgrade/`, `internal/cli/sdk/client.go`, and `cmd/cortex/commands/`. Custom error types themselves are well-designed (StateError, TmuxError, etc.) with proper `Is*()` checker functions.

### 11. Inconsistent logging: 2 instances of `log.Printf` in `slog` codebase
- `internal/daemon/api/tickets.go:818,832` — uses `log.Printf` instead of structured `slog.Warn()`
- Rest of codebase consistently uses `log/slog`

### 12. Missing OSS standard files
- `CODE_OF_CONDUCT.md` — not present
- `.github/` templates (ISSUE_TEMPLATE, PR_TEMPLATE) — not present
- Consider adding for community contribution

### 13. Naming: `DocsStoreManager` stutters
- `internal/daemon/api/docs_store_manager.go:16` — `DocsStoreManager` is verbose
- `StoreManager` (for tickets) also slightly ambiguous in `api` package context
- Minor naming inconsistency: `CreateComment()` in `internal/storage/comment.go:51` vs `New*()` pattern everywhere else

### 14. `TmuxChecker` interface misplaced in types package
- `internal/types/convert.go:13` — `TmuxChecker` interface is a narrow concern placed in the generic `types` package
- Should move to a more specific location

### 15. CLI commands under-documented
- Only 30-40% of `cmd/cortex/commands/` and `cmd/cortexd/commands/` files have godoc comments
- Core packages (api, sdk, ticket, mcp) have good coverage

### 16. CONFIG_DOCS.md buried in agent defaults
- Located at `internal/install/defaults/claude-code/CONFIG_DOCS.md`
- Hard to discover — consider linking from root docs or duplicating to `docs/`

### 17. Magic timeout values scattered
- Server timeouts: `15s read`, `0 write`, `60s idle` in `api/server.go`
- Client timeout: `10s` in `cli/sdk/client.go`
- Hook timeout: `5s` in `commands/hook.go`
- Autostart retries: `1s, 2s, 5s` in `autostart/autostart.go`
- Consider extracting to named constants with rationale comments

### 18. `.gitignore` could be more comprehensive
Missing patterns: `.env*`, `*.pid`, `.cortex/logs/`