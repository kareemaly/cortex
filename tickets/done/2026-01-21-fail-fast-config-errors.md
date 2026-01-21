# Fail Fast on Configuration Errors

Remove silent fallbacks and fail fast when configuration is missing or invalid.

## Context

Multiple places in the codebase silently fall back to defaults when configuration is missing, causing data to go to wrong locations or operations to happen in wrong contexts without any error.

## Issues to Fix

### 1. MCP Server Tickets Directory (CRITICAL)

**File:** `internal/daemon/mcp/server.go:57-65`

**Current:**
```go
ticketsDir := cfg.TicketsDir
if ticketsDir == "" {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return nil, err
    }
    ticketsDir = filepath.Join(homeDir, ".cortex", "tickets")
}
```

**Fix:** Derive from `ProjectPath` or fail:
```go
ticketsDir := cfg.TicketsDir
if ticketsDir == "" {
    if cfg.ProjectPath != "" {
        ticketsDir = filepath.Join(cfg.ProjectPath, ".cortex", "tickets")
    } else {
        return nil, fmt.Errorf("MCP server requires CORTEX_PROJECT_PATH or CORTEX_TICKETS_DIR to be set")
    }
}
```

### 2. CORTEX_TMUX_SESSION Not Read (CRITICAL)

**File:** `cmd/cortexd/commands/mcp.go:48-49`

**Current:** Only reads `CORTEX_TICKETS_DIR` and `CORTEX_PROJECT_PATH`

**Fix:** Also read `CORTEX_TMUX_SESSION`:
```go
ticketsDir := os.Getenv("CORTEX_TICKETS_DIR")
projectPath := os.Getenv("CORTEX_PROJECT_PATH")
tmuxSession := os.Getenv("CORTEX_TMUX_SESSION")

cfg := &mcp.Config{
    TicketID:    ticketID,
    TicketsDir:  ticketsDir,
    ProjectPath: projectPath,
    TmuxSession: tmuxSession,
}
```

### 3. TmuxSession Silent Default (CRITICAL)

**File:** `internal/daemon/mcp/server.go:68-69`

**Current:**
```go
if cfg.TmuxSession == "" {
    cfg.TmuxSession = "cortex"
}
```

**Fix:** Fail if spawning sessions without tmux context:
```go
// Remove the default - let it stay empty
// Validate in handleSpawnSession instead:
if s.config.TmuxSession == "" {
    return nil, SpawnSessionOutput{
        Success: false,
        Message: "cannot spawn session: CORTEX_TMUX_SESSION not configured",
    }, nil
}
```

### 4. Child Sessions Missing CORTEX_TMUX_SESSION (CRITICAL)

**File:** `internal/daemon/mcp/tools_architect.go:394-400`

**Current:** Only passes `CORTEX_TICKETS_DIR` and `CORTEX_PROJECT_PATH`

**Fix:** Also pass `CORTEX_TMUX_SESSION`:
```go
if s.config.TicketsDir != "" {
    mcpConfig.MCPServers["cortex"].Env["CORTEX_TICKETS_DIR"] = s.config.TicketsDir
}
if s.config.ProjectPath != "" {
    mcpConfig.MCPServers["cortex"].Env["CORTEX_PROJECT_PATH"] = s.config.ProjectPath
}
if s.config.TmuxSession != "" {
    mcpConfig.MCPServers["cortex"].Env["CORTEX_TMUX_SESSION"] = s.config.TmuxSession
}
```

### 5. Daemon Config Silent Default (HIGH)

**File:** `internal/daemon/config/config.go:33-35`

**Current:**
```go
homeDir, err := os.UserHomeDir()
if err != nil {
    return cfg, nil  // Silent failure
}
```

**Fix:** Return the error:
```go
homeDir, err := os.UserHomeDir()
if err != nil {
    return nil, fmt.Errorf("failed to get home directory: %w", err)
}
```

### 6. Install Command Discards Error (HIGH)

**File:** `cmd/cortex/commands/install.go:76`

**Current:**
```go
homeDir, _ := os.UserHomeDir()
```

**Fix:** Handle the error:
```go
homeDir, err := os.UserHomeDir()
if err != nil {
    homeDir = ""  // Acceptable fallback for display purposes only
}
```

Or just don't do the replacement if it fails (this is display-only, not critical).

## Verification

```bash
# Test MCP fails without project path
CORTEX_PROJECT_PATH= cortexd mcp
# Should error: "MCP server requires CORTEX_PROJECT_PATH..."

# Test spawn fails without tmux session
# (from within MCP, try spawnSession tool without CORTEX_TMUX_SESSION)
# Should return error message about missing tmux session config

# Run existing tests
make test
make test-integration
```

## Notes

- The `cortexd serve` command can keep its fallback behavior (it's the global daemon)
- Project config returning defaults when `cortex.yaml` is missing is acceptable (explicit project init creates the file)
- Focus on MCP server config since that's where data corruption can occur

## Implementation

### Commits Pushed

- `eaff9f8` feat: fail fast on MCP configuration errors instead of silent fallbacks

### Key Files Changed

- `internal/daemon/mcp/server.go` - TicketsDir now derived from ProjectPath or requires explicit setting; removed TmuxSession default
- `cmd/cortexd/commands/mcp.go` - Added reading of CORTEX_TMUX_SESSION environment variable
- `internal/daemon/mcp/tools_architect.go` - Added spawn-time validation for TmuxSession; pass CORTEX_TMUX_SESSION to child sessions
- `internal/daemon/config/config.go` - Return error on UserHomeDir failure instead of silent fallback
- `cmd/cortex/commands/install.go` - Explicit error handling with comment explaining acceptable fallback for display
- `internal/daemon/mcp/server_test.go` - Updated TestNewServerNilConfig to expect error; added TestNewServerWithProjectPath
- `internal/daemon/mcp/tools_test.go` - Added TmuxSession to test config fixture

### Important Decisions

- TmuxSession validation happens at spawn time (in handleSpawnSession) rather than at server creation, since architect sessions that don't spawn are still valid without it
- Install command keeps fallback for homeDir since it's display-only (path prettification with ~)
- Updated doc comments in Config struct to reflect new behavior

### Scope Changes

None - implemented all six changes as specified in the ticket.
