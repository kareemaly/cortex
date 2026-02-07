---
id: 0e8358c8-1104-45c8-b53d-3e5af7ba5794
title: Fix CI Failures
type: ""
created: 2026-01-21T19:11:53Z
updated: 2026-01-21T19:11:53Z
---
GitHub Actions CI is failing on both `build` and `lint` jobs.

## Issues

### 1. Test Failure - `TestHandleSpawnSession`

The test calls `binpath.FindCortexd()` which fails in CI because there's no `cortexd` binary.

**Error:**
```
cortexd not found: exec: "cortexd": executable file not found in $PATH
```

**Fix:** Mock the binpath lookup in the test or make it injectable.

### 2. Lint Failure - Invalid golangci-lint config

The `.golangci.yml` has fields not supported by the latest golangci-lint.

**Error:**
```
additional properties 'default' not allowed
additional properties 'version', 'formatters' not allowed
```

**Fix:** Update `.golangci.yml` to use valid schema.

## Verification

```bash
# Check CI status after push
gh run list --repo kareemaly/cortex --limit 1
```

## Implementation

### Commits Pushed

- `4099c67` fix: make cortexd path injectable for testing
- `ad4fb52` fix: pin golangci-lint to v2.8.0 for v2 config compatibility
- `3b04b02` ci: update golangci-lint-action to v7 for v2 support

### Key Files Changed

- `internal/daemon/mcp/server.go` - Added `CortexdPath` field to Config struct
- `internal/daemon/mcp/tools_architect.go` - Updated `handleSpawnSession` to use injected path if provided
- `internal/daemon/mcp/tools_test.go` - Added mock cortexd path to test setup
- `.github/workflows/ci.yml` - Pinned golangci-lint version to v2.8.0, updated action to v7

### Important Decisions

- Followed established dependency injection pattern (like `TmuxManager`) for testability
- Used `/mock/cortexd` as the test path since the actual binary isn't needed when tmux is also mocked
- Pinned golangci-lint to v2.8.0 because `version: latest` in the action still defaults to v1.x, which doesn't support v2 config syntax
- Updated golangci-lint-action from v6 to v7 because v6 doesn't support golangci-lint v2.x versions

### Scope Changes

- Both test and lint issues from the original ticket are now fixed
- Additional fix required: golangci-lint-action@v6 doesn't support v2 versions, needed v7