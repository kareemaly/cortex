---
id: 93a35741-cb64-402a-81c8-44a37067b230
author: claude
type: review_requested
created: 2026-02-04T13:00:18.831735Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/88d421fd-e96b-43f0-967b-b06c820a571e
---
## Summary

Fixed CI test failures where 9 tests in `internal/daemon/mcp/tools_test.go` failed because spawn operations tried to find `cortexd` in PATH, which is not available in CI.

## Changes Made

1. **internal/daemon/api/deps.go** - Added `CortexdPath string` field to `Dependencies` struct

2. **internal/daemon/api/tickets.go** - Pass `h.deps.CortexdPath` to `spawn.OrchestrateDeps` in the Spawn handler

3. **internal/daemon/api/architect.go** - Pass `h.deps.CortexdPath` to `spawn.Dependencies` when creating the spawner

4. **internal/daemon/mcp/tools_test.go** - Inject mock path `/mock/cortexd` in both test setup functions (`setupArchitectWithDaemon` and `setupTicketSession`)

## Verification

- `make test` - All tests pass
- `make lint` - No lint issues (0 issues)
- `go mod tidy` - Completed successfully

## Notes

The fix doesn't change production behavior since `CortexdPath` defaults to empty string, which triggers the existing `binpath.FindCortexd()` auto-discovery. Tests now inject a mock path to bypass the PATH lookup.