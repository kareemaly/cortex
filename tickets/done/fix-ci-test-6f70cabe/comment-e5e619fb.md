---
id: e5e619fb-3c7d-4a68-9dac-43578017fba8
author: claude
type: done
created: 2026-02-04T13:02:33.59171Z
---
## Summary

Fixed CI test failures where 9 tests in `internal/daemon/mcp/tools_test.go` failed because spawn operations tried to find `cortexd` in PATH, which is not available in CI environments.

## Root Cause

Tests created HTTP test servers with `api.Dependencies` without a `CortexdPath`, causing spawn operations to fall back to `exec.LookPath("cortexd")` which fails in CI.

## Solution

Added `CortexdPath string` field to `api.Dependencies` and propagated it through spawn operations. Tests now inject a mock path `/mock/cortexd` to bypass the PATH lookup.

## Files Changed

1. `internal/daemon/api/deps.go` - Added `CortexdPath string` field
2. `internal/daemon/api/tickets.go` - Pass `CortexdPath` to `spawn.OrchestrateDeps`
3. `internal/daemon/api/architect.go` - Pass `CortexdPath` to `spawn.Dependencies`
4. `internal/daemon/mcp/tools_test.go` - Inject mock path in both test setup functions

## Verification

- All tests pass (`make test`)
- No lint issues (`make lint`)
- Changes merged to main and pushed to origin