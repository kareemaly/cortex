---
id: c2ad7e8e-99a3-4600-a36d-ccb59d60a728
title: Add missing test coverage for SDK client, upgrade, and ticket handlers
type: work
tags:
    - oss-readiness
    - testing
created: 2026-02-08T13:10:59.108557Z
updated: 2026-02-08T13:34:44.892934Z
---
Critical packages have zero or insufficient test coverage. Add unit tests to the highest-risk areas.

## Priority targets

### 1. `internal/cli/sdk/client.go` — ZERO tests
The primary HTTP client used by all CLI commands. Should have unit tests using httptest.Server to verify:
- Request construction (headers, paths, query params)
- Response parsing for all methods
- Error handling (HTTP errors, network errors, malformed JSON)
- Project header injection

### 2. `internal/upgrade/` — ZERO tests (4 files)
Self-update logic involving binary replacement. High risk of breakage. Test:
- Version comparison logic
- Binary download and replacement flow (mock HTTP)
- Backup creation and rollback
- Permission handling

### 3. `internal/daemon/api/tickets.go` — 964 lines, no unit tests
Largest handler file. Only covered indirectly by integration tests. Add unit tests for:
- Individual handler functions using httptest
- Edge cases in request parsing
- Error paths

## Existing test patterns to follow
- Table-driven tests with `t.Run()` subtests
- `t.TempDir()` for filesystem cleanup
- `httptest.Server` for HTTP mocking (see existing patterns in codebase)
- Integration tests gated with `//go:build integration`