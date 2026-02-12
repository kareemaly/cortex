---
id: 6fa5e6b0-e4e5-4c92-be5d-44a29080c06a
title: 'API cleanup: centralize daemon URL, consistent errors, deduplicate types'
type: work
tags:
    - oss-readiness
    - cleanup
created: 2026-02-08T13:10:50.119596Z
updated: 2026-02-08T13:43:41.874247Z
---
Clean up the HTTP API layer for consistency and maintainability.

## 1. Centralize hardcoded daemon URL
Currently `http://localhost:4200` is hardcoded in 4 separate locations:
- `internal/cli/sdk/client.go:17` — `const defaultBaseURL`
- `cmd/cortexd/commands/hook.go:16` — `const defaultDaemonURL`
- `internal/core/spawn/config.go:56` — `CORTEX_DAEMON_URL`
- `internal/daemon/mcp/server.go:101` — fallback `cfg.DaemonURL`

Define one canonical source (likely derived from daemon config port + bind address) and have all consumers read from that or from `CORTEX_DAEMON_URL` env var.

## 2. Consistent HTTP error responses
Mixed use of raw `http.Error()` (plain text) and `writeError()` (JSON) across API handlers:
- `internal/daemon/api/projects.go:40,46,51,56,69,106` — uses `http.Error()`
- `internal/daemon/api/handlers.go:49` — uses `http.Error()`
- `internal/daemon/api/events.go:23` — uses `http.Error()`

Replace all `http.Error()` calls in API handlers with the existing `writeError()` helper so consumers always get JSON error responses.

## 3. Deduplicate shared types
6 type definitions are duplicated between SDK client and API:
- `HealthResponse` — `cli/sdk/client.go:85` + `daemon/api/handlers.go:12`
- `AddCommentResponse` — `cli/sdk/client.go:532` + `daemon/api/types.go:79`
- `RequestReviewResponse` — `cli/sdk/client.go:538` + `daemon/api/types.go:92`
- `ConcludeSessionResponse` — `cli/sdk/client.go:545` + `daemon/api/types.go:104`
- `ProjectResponse` — `cli/sdk/client.go:473` + `daemon/api/projects.go:22`
- `ResolvePromptResponse` — `cli/sdk/client.go:1068` + `daemon/api/prompts.go:21`

Move these to `internal/types/` (which already exists) and import from both places.

## 4. Stray log.Printf calls
- `internal/daemon/api/tickets.go:818,832` — uses `log.Printf` instead of `slog.Warn()`
- Rest of codebase consistently uses `log/slog`