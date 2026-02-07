---
id: d9f858d7-a5fb-4cc8-bf29-f93b09c5ccc6
author: claude
type: review_requested
created: 2026-02-03T08:53:00.819921Z
action:
    type: git_diff
    args:
        commit: 8b9f088
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/622302ce-1376-4916-88cc-1a1518273e33
---
## Summary

Added `getCortexConfigDocs` MCP tool for architect sessions that returns the embedded CONFIG_DOCS.md content for the project's configured agent type.

## Changes

### `internal/install/embed.go`
- Added `GetConfigDocs(configName string)` function to read embedded CONFIG_DOCS.md files

### `internal/daemon/mcp/types.go`
- Added `GetCortexConfigDocsInput` struct (empty, tool takes no parameters)
- Added `GetCortexConfigDocsOutput` struct with `Content` and `ConfigName` fields

### `internal/daemon/mcp/tools_architect.go`
- Registered `getCortexConfigDocs` tool in `registerArchitectTools()`
- Implemented `handleGetCortexConfigDocs` handler that:
  - Determines config name from architect agent type (e.g., "claude" → "claude-code")
  - First checks resolved extend path for user customization
  - Falls back to embedded file if not found in extend path

## Verification

- `make build` - passes
- `make lint` - passes with no issues
- `make test` - all unit tests pass