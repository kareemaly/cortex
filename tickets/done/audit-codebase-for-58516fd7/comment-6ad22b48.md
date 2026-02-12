---
id: 6ad22b48-a241-47e2-aa29-71214fbe885d
author: claude
type: comment
created: 2026-02-08T13:04:59.00703Z
---
## P1 - Should Fix (Security & Code Quality)

### 4. Hardcoded daemon URL in 4+ locations
- **Impact:** Can't connect to remote daemon without code changes; violates DRY
- **Locations:**
  - `internal/cli/sdk/client.go:17` — `const defaultBaseURL = "http://localhost:4200"`
  - `cmd/cortexd/commands/hook.go:16` — `const defaultDaemonURL = "http://localhost:4200"`
  - `internal/core/spawn/config.go:56` — `CORTEX_DAEMON_URL = "http://localhost:4200"`
  - `internal/daemon/mcp/server.go:101` — fallback `cfg.DaemonURL = "http://localhost:4200"`
- **Action:** Read from daemon config or `CORTEX_DAEMON_URL` env consistently. Define one canonical source.

### 5. Inconsistent HTTP error response format
- **Impact:** API consumers get mixed JSON vs plain text errors
- **Locations:**
  - `internal/daemon/api/projects.go:40,46,51,56,69,106` — uses raw `http.Error()` with mixed JSON/text
  - `internal/daemon/api/handlers.go:49` — plain text error
  - `internal/daemon/api/events.go:23` — plain text error
- **Action:** Replace all `http.Error()` in API handlers with the existing `writeError()` helper for consistent JSON error responses.

### 6. Duplicate type definitions across SDK and API
- **Impact:** Maintenance burden — changes in one place must be mirrored in other
- **Duplicated types (6 pairs):**
  - `HealthResponse` — `cli/sdk/client.go:85` + `daemon/api/handlers.go:12`
  - `AddCommentResponse` — `cli/sdk/client.go:532` + `daemon/api/types.go:79`
  - `RequestReviewResponse` — `cli/sdk/client.go:538` + `daemon/api/types.go:92`
  - `ConcludeSessionResponse` — `cli/sdk/client.go:545` + `daemon/api/types.go:104`
  - `ProjectResponse` — `cli/sdk/client.go:473` + `daemon/api/projects.go:22`
  - `ResolvePromptResponse` — `cli/sdk/client.go:1068` + `daemon/api/prompts.go:21`
- **Action:** Move shared types to `internal/types/` (which already exists for other response types) and re-export.

### 7. File permissions too permissive for sensitive files
- **Locations:**
  - `internal/daemon/autostart/pidfile.go:78` — PID file `0644` (should be `0600`)
  - `internal/daemon/config/config.go:90` — settings.yaml `0644` (should be `0600`)
  - `internal/upgrade/binary.go:63,78,107` — backup binaries `0755` (should be `0700`)
- **Action:** Use `0600` for user config files, `0700` for executable backups.

### 8. SSE event stream silently drops errors
- **Location:** `internal/daemon/api/events.go:47-49`
```go
if err != nil {
    continue  // JSON marshal error silently skipped, no logging
}
_, _ = fmt.Fprintf(w, "data: %s\n\n", data)  // Write error ignored
```
- **Action:** Log marshaling failures; detect write errors to clean up dead connections.