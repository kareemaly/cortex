---
id: 77c8df55-d22a-47a9-8b86-eb392b55091f
title: 'Security hardening: localhost binding, file permissions, SSE error handling'
type: work
tags:
    - oss-readiness
    - security
created: 2026-02-08T13:10:37.707092Z
updated: 2026-02-08T13:19:06.72648Z
---
Bundle of security fixes needed before open source release.

## 1. Daemon binds to 0.0.0.0 — change to 127.0.0.1 by default
- **Location:** `internal/daemon/api/server.go:113-118`
- Current: `Addr: fmt.Sprintf(":%d", port)` binds to all interfaces
- Change to: `fmt.Sprintf("127.0.0.1:%d", port)` by default
- Add a config option (e.g., `daemon.bind_address` in settings.yaml) for users who explicitly need `0.0.0.0` (remote VM deployments)

## 2. File permissions too permissive
- `internal/daemon/autostart/pidfile.go:78` — PID file uses `0644`, should be `0600`
- `internal/daemon/config/config.go:90` — settings.yaml uses `0644`, should be `0600`
- `internal/upgrade/binary.go:63,78,107` — backup binaries use `0755`, should be `0700`

## 3. SSE event stream silently drops errors
- **Location:** `internal/daemon/api/events.go:47-49`
- JSON marshal errors are silently `continue`d — should log with `slog.Warn()`
- Write errors to `fmt.Fprintf` are ignored — should detect and clean up dead connections