---
id: aebcd12c-bb38-45f3-8ca3-194bd71ade30
author: claude
type: done
created: 2026-02-15T11:39:56.214607Z
---
## Summary

Added automatic SSE reconnection with exponential backoff and periodic poll fallback to all four TUI views.

### Problem
When the SSE connection dropped (daemon restart, network hiccup, HTTP timeout), all TUI views silently lost live updates permanently. Users had to manually press `r` to refresh.

### Solution
Implemented reconnection logic across 4 files (323 lines added, 18 removed):

- **`internal/cli/tui/kanban/model.go`** — Template implementation with reconnect logic
- **`internal/cli/tui/docs/model.go`** — Mirrors kanban, uses `loadDocs()` for reload
- **`internal/cli/tui/ticket/model.go`** — Same pattern, no logBuf, ticket-filtering waitForEvent, embedded mode support
- **`internal/cli/tui/dashboard/model.go`** — Per-project SSE with `sseBackoffs` map

### Key mechanics
1. **Exponential backoff**: 2s → 4s → 8s → ... → 30s cap on reconnect attempts
2. **60s poll fallback**: Safety-net periodic data refresh regardless of SSE state
3. **Stale disconnect guard**: Prevents spurious reconnect loops when a new SSE connection replaces an old one (single-connection views use `sseConnected` bool; dashboard uses `sseChannels[path]` check)
4. **Clean reconnect**: `sseConnectedMsg` handler cancels old context first, resets backoff, reloads data

### Decisions
- No SDK or server-side changes needed — all logic is client-side in TUI models
- No documentation updates needed — changes are internal TUI implementation details
- Each package defines its own `nextBackoff()` helper to avoid cross-package dependencies

### Verification
- `make build` — clean compilation
- `make lint` — 0 issues
- `make test` — all unit tests pass
- Pre-push hooks passed (lint + build)