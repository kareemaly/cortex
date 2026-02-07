---
id: 71782f5c-5c51-4666-9cb0-d45fc4894df0
title: Fix unit tests to not create real tmux sessions
type: ""
created: 2026-01-21T09:18:41Z
updated: 2026-01-21T09:18:41Z
---
`make test` should run pure unit tests without side effects. Tests should not create real tmux sessions or spawn actual agents.

## Problem

Some tests may be creating actual tmux sessions or attempting to run real commands, which:
- Causes test pollution (leftover sessions)
- Makes tests flaky (depends on tmux being installed)
- Slows down test suite
- Can interfere with user's running tmux sessions

## Requirements

1. **Audit existing tests**
   - Check `internal/tmux/*_test.go` for real tmux calls
   - Check `internal/daemon/mcp/*_test.go` for spawn operations
   - Identify any tests that shell out to real commands

2. **Mock external dependencies**
   - Tmux operations should be mockable (interface-based)
   - Agent spawning should be mocked in tests
   - File system operations should use temp directories

3. **Keep tests fast and isolated**
   - No network calls
   - No real process spawning
   - No persistent state between tests

4. **Consider integration test separation**
   - If we need real tmux tests, separate them: `make test-integration`
   - Unit tests (`make test`) should always be safe to run

## Verification

```bash
# Kill any existing tmux sessions
tmux kill-server 2>/dev/null || true

# Run tests
make test

# Verify no tmux sessions were created
tmux list-sessions 2>&1 | grep -q "no server running" && echo "PASS: No sessions created"
```

## Notes

- Tests should pass even if tmux is not installed
- Consider using interfaces for tmux.Manager to enable mocking

## Implementation

### Commits Pushed

- `891785d` fix: add TmuxRunner interface to prevent tests from creating real tmux sessions
- `1ba5077` Merge branch 'ticket/2026-01-21-unit-tests-no-side-effects'

### Key Files Changed

- `internal/tmux/tmux.go` - Added `TmuxRunner` interface, `execRunner` default implementation, and `NewManagerWithRunner()` for dependency injection
- `internal/tmux/session.go` - Updated `AttachSession()` to use runner instead of direct exec.Command
- `internal/tmux/mock_runner.go` (new) - Created `MockRunner` test implementation with configurable callbacks
- `internal/daemon/mcp/server.go` - Added `TmuxManager` field to `Config` and `Server` structs
- `internal/daemon/mcp/tools_architect.go` - Updated `handleSpawnSession()` to use injected manager if available
- `internal/daemon/mcp/tools_test.go` - Added `setupTestServerWithMockTmux()` helper and updated spawn session tests

### Important Decisions

- Followed existing pattern from `internal/lifecycle/hooks.go` which uses `CommandRunner` interface with `NewExecutorWithRunner()` for testing
- The `TmuxRunner` interface has two methods: `Run()` for non-interactive commands and `RunInteractive()` for attach/switch operations
- Mock runner returns sensible defaults (e.g., window index "1" for new-window, empty success for most operations)
- Integration tests remain unchanged (`internal/tmux/integration_test.go` with `//go:build integration`)

### Scope Changes

None - implemented as originally planned.