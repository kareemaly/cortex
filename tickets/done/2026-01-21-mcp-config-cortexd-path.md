# Fix cortexd path in MCP config generation

MCP config generation hardcodes `"command": "cortexd"` but cortexd may not be in PATH, causing MCP connection failures.

## Symptoms

- Claude shows "MCP not connected"
- Claude hangs on startup waiting for MCP server
- MCP server fails to start because `cortexd` command not found

## Root Cause

Two locations hardcode the command:
- `cmd/cortex/commands/architect.go:151` - `Command: "cortexd"`
- `internal/daemon/mcp/tools_architect.go:369` - `Command: "cortexd"`

## Requirements

1. **Find cortexd binary path dynamically**
   - Use `os.Executable()` to get the current binary path
   - Derive cortexd path from cortex path (same directory)
   - Example: if cortex is at `/Users/foo/.local/bin/cortex`, cortexd is at `/Users/foo/.local/bin/cortexd`

2. **Update both MCP config generators**
   - `cmd/cortex/commands/architect.go` - `generateArchitectMCPConfig()`
   - `internal/daemon/mcp/tools_architect.go` - `spawnSession()`

3. **Consider edge cases**
   - When running from `go run` during development
   - When binary name doesn't follow pattern (fallback to PATH lookup)

## Verification

```bash
make build
make lint
make test

# Install to non-PATH location
cp bin/cortex bin/cortexd ~/.local/bin/

# Manual test (from tmux, after fixing nested session issue)
cd ~/projects/test-cortex
cortex architect
# Should spawn with working MCP tools
```

## Notes

- This is blocking MCP functionality for installed binaries
- A helper function to find cortexd path could be shared between both locations

## Implementation

### Commits Pushed
- `3badbb2` fix: use absolute path for cortexd in MCP config generation
- `4e46f2b` Merge branch 'ticket/2026-01-21-mcp-config-cortexd-path'

### Key Files Changed
- `internal/binpath/binpath.go` (new) - Shared helper function `FindCortexd()` that derives cortexd path from current executable or falls back to PATH lookup
- `cmd/cortex/commands/architect.go` - Updated `generateArchitectMCPConfig()` to use `binpath.FindCortexd()`
- `internal/daemon/mcp/tools_architect.go` - Updated `handleSpawnSession()` to use `binpath.FindCortexd()` with proper error handling and session cleanup

### Important Decisions
- Created a shared `binpath` package rather than duplicating logic in both locations
- Strategy: first check if running as cortex/cortexd and derive sibling path, then fall back to PATH lookup
- Added session cleanup in `handleSpawnSession()` if cortexd lookup fails to avoid orphaned sessions

### Scope Changes
- None - implemented as specified in the plan
