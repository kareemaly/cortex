---
id: c8b0fba1-57b6-49f0-8c76-2b431732e518
title: Spawn Error Handling
type: ""
created: 2026-01-24T15:55:41Z
updated: 2026-01-24T15:55:41Z
---
## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Problem

1. **Cleanup errors silently ignored** - `internal/core/spawn/spawn.go` cleanup operations use `_ = operation()` without logging:
   ```go
   _ = wm.Remove(*worktreePath, *featureBranch)
   _ = RemoveMCPConfig(mcpConfigPath)
   _ = RemoveSettingsConfig(settingsPath)
   ```
   These appear at lines 165, 301, 321, 322, 343, 530, 533, 540.

2. **Missing validation** - `validateSpawnRequest()` doesn't check:
   - `ProjectPath` exists on filesystem
   - `TmuxSession` contains valid characters

## Requirements

- Log cleanup errors as warnings instead of silently ignoring
- Add ProjectPath existence validation
- Add TmuxSession name validation

## Implementation

### Commits

- `9a416a3` feat: improve spawn error handling and input validation
- `f6d65f3` Merge branch 'ticket/2026-01-24-spawn-error-handling'

### Key Files Changed

- `internal/core/spawn/spawn.go` - Added Logger to Dependencies, logWarn helper, cleanup error logging, and input validations
- `internal/core/spawn/spawn_test.go` - Added tests for validateTmuxName and ProjectPath validation
- `internal/daemon/api/tickets.go` - Pass logger to spawner
- `internal/daemon/mcp/server.go` - Added Logger field to Config struct
- `internal/daemon/mcp/tools_architect.go` - Pass logger to spawner

### Decisions

- Used nil-safe `logWarn()` helper method to avoid nil pointer issues when Logger is not provided
- TmuxSession validation rules: max 128 chars, no leading hyphen, no colons/periods (tmux delimiters), alphanumeric/underscores/hyphens only
- ProjectPath validation only checks existence when path is non-empty (empty path is allowed for some use cases)