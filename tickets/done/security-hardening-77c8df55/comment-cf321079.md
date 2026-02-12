---
id: cf321079-0dfc-40e2-bfd7-d6f9dbf25382
author: claude
type: review_requested
created: 2026-02-08T13:17:30.101814Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/afbdd067-b497-425f-8544-4fc67fbf73b0
        commit: a3496d6
---
## Security Hardening Changes

### 1. Localhost binding by default
- Added `BindAddress` field to `Config` struct with default `"127.0.0.1"` — daemon no longer binds to all interfaces
- Updated `NewServer` signature to accept `bindAddress` parameter
- Wired through from `serve.go` using `cfg.BindAddress`
- Users can override with `bind_address: "0.0.0.0"` in `settings.yaml` for remote VM deployments

### 2. Tightened file permissions
- PID file (`pidfile.go`): `0644` → `0600`
- `settings.yaml` (`config.go`): `0644` → `0600`
- Backup directory (`binary.go`): `0755` → `0700`
- Backup binary (`binary.go`): `0755` → `0700`
- Replaced binary (`binary.go`): `0755` → `0700`

### 3. SSE error handling
- JSON marshal errors now logged with `slog.Warn` before continuing
- Write errors to `fmt.Fprintf` now detected, logged, and cause `return` (deferred `unsubscribe()` handles cleanup)

### 4. Tests
- Updated config round-trip test to verify `BindAddress` persists correctly

### Verification
- `make build` — compiles successfully
- `make test` — all tests pass
- `make lint` — 0 issues