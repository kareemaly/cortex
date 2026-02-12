---
id: 9805378c-b01f-49f1-ad33-8cead7f3987f
author: claude
type: done
created: 2026-02-08T13:19:06.724125Z
---
## Security Hardening — Completed

### Changes (9 files, 2 commits)

**1. Localhost binding by default** (`config.go`, `server.go`, `serve.go`)
- Added `BindAddress` config field defaulting to `"127.0.0.1"` — daemon no longer binds to all network interfaces
- Users can override with `bind_address: "0.0.0.0"` in `settings.yaml` for remote VM deployments
- Updated `NewServer` signature to accept `bindAddress` parameter

**2. Tightened file permissions** (`pidfile.go`, `config.go`, `binary.go`)
- PID file: `0644` → `0600`
- `settings.yaml`: `0644` → `0600`
- Backup directory: `0755` → `0700`
- Backup binary: `0755` → `0700`
- Replaced binary: `0755` → `0700`

**3. SSE error handling** (`events.go`)
- JSON marshal errors logged with `slog.Warn` before continuing
- Write errors detected, logged, and trigger `return` (deferred `unsubscribe()` handles cleanup)

**4. Documentation** (`README.md`, `CLAUDE.md`)
- Updated README config example to show `bind_address` field
- Updated CLAUDE.md global config description to mention bind address

**5. Tests** (`config_test.go`)
- Added `BindAddress` assertion to round-trip test

### Verification
- `make build` — compiles
- `make test` — all tests pass
- `make lint` — 0 issues
- Pre-push hooks passed