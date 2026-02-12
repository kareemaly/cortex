---
id: 9895221e-0e91-45d6-b433-e4e137437f48
author: claude
type: review_requested
created: 2026-02-08T13:40:59.748542Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/f898bb46-cd75-4feb-80db-f4afd1b31acb
        commit: 95d779eaf5f3bfc869c77a7538a60fe424a924f1
---
## API Cleanup: Centralize Daemon URL, Consistent Errors, Deduplicate Types

### Changes across 15 files:

**1. Deduplicate shared types** (internal/types/response.go, api/types.go, sdk/client.go, api/handlers.go, api/projects.go, api/prompts.go)
- Added 7 shared response types to `internal/types/response.go`: HealthResponse, ProjectTicketCounts, ProjectResponse, AddCommentResponse, RequestReviewResponse, ConcludeSessionResponse, ResolvePromptResponse
- Added type aliases in `api/types.go` and `sdk/client.go` re-export blocks
- Removed 7 duplicate struct definitions from `sdk/client.go` and 6 from various `api/` files

**2. Centralize hardcoded daemon URL** (daemon/config/config.go, sdk/client.go, hook.go, spawn/config.go, mcp/server.go, serve.go, 2 test files)
- Added `DefaultPort = 4200`, `DefaultDaemonURL = "http://localhost:4200"`, and `DaemonURL()` helper to `daemon/config`
- Updated all 4 hardcoded URL locations to use the constant
- SDK client and hook now check `CORTEX_DAEMON_URL` env var with fallback to `DefaultDaemonURL`
- `MCPConfigParams` gains a `DaemonURL` field with fallback to the constant
- Tests updated to use `daemonconfig.DefaultDaemonURL`

**3. Consistent HTTP error responses** (api/projects.go, api/handlers.go, api/events.go)
- Replaced all 8 `http.Error()` calls with `writeError()` for consistent JSON error responses

**4. Replace stray log.Printf** (api/tickets.go)
- Replaced 2 `log.Printf` calls with `h.deps.Logger.Warn` structured logging
- Removed unused `"log"` import

### Verification
- `make build` ✓
- `make test` ✓ (all unit tests pass)
- `make lint` ✓ (0 issues)