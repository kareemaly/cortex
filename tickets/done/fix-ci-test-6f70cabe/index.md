---
id: 6f70cabe-709a-4409-9ca9-50eec40b757f
title: Fix CI test failures - cortexd not in PATH
type: work
created: 2026-02-04T12:46:48.922857Z
updated: 2026-02-04T13:02:33.59299Z
---
## Problem

CI tests fail because spawn-related tests in `internal/daemon/mcp/tools_test.go` require the `cortexd` binary to be in PATH, but it's not built/installed during `make test`.

**Error:**
```
spawn: cortexd not found: exec: "cortexd": executable file not found in $PATH
```

**9 failing tests:**
- TestHandleSpawnSession
- TestHandleSpawnSessionAutoMovesToProgress
- TestHandleSpawnSession_StateNormal_ModeNormal
- TestHandleSpawnSession_StateOrphaned_ModeResume
- TestHandleSpawnSession_StateOrphaned_ModeFresh
- TestHandleSpawnSession_StateEnded_ModeNormal
- TestHandleSpawnSession_StateEnded_ModeFresh
- TestHandleSpawnSession_DefaultMode

## Requirements

Fix the tests so they pass in CI without requiring installed binaries. Options to consider:

1. **Mock the binary lookup** — inject the binary path resolver so tests can provide a mock
2. **Build before test** — update Makefile to build binaries before running tests
3. **Skip when unavailable** — skip spawn tests if cortexd not found (less ideal)
4. **Refactor for testability** — make spawn orchestration accept interfaces that can be mocked

Also run `go mod tidy` to fix the diagnostic about `github.com/charmbracelet/x/ansi`.

## Verification

- `make test` passes locally
- CI workflow passes