# Core Spawn Package

Create shared spawn logic used by both MCP tools and HTTP API handlers.

## Context

This is a fresh project with no users. No backward compatibility needed. Breaking changes are fine. Do not accumulate tech debt.

## Package Location

`internal/core/spawn/`

## Responsibilities

### State Detection
- Check if tmux window exists for a session
- Determine session state: `Normal` (no session), `Active` (session + window), `Orphaned` (session but no window), `Ended` (session.EndedAt set)

### Spawn Operations
- `Spawn(req)` - create new session, generate MCP config, spawn tmux window
- `Resume(req)` - resume orphaned session with `claude --resume {sessionID}`
- `Fresh(req)` - clear existing session, spawn new one

### Shared Logic (extract from current MCP tools)
- MCP config JSON generation (currently in `tools_architect.go`)
- Prompt template loading (use `internal/prompt/` package)
- Shell escaping for prompts
- Tmux window spawning
- Session creation with `store.SetSession()`
- Error cleanup/rollback on failure

## Spawn Request

Should support both architect and ticket agent spawning:
- Architect: no ticket, uses architect.md prompt
- Ticket Agent: has ticketID, uses ticket-agent.md prompt

## Used By

- `internal/daemon/mcp/tools_architect.go` - spawnSession tool
- `internal/daemon/api/` - spawn endpoints (future ticket)
- `cmd/cortex/commands/architect.go` - architect spawning (future ticket)

## Verification

```bash
make lint
make test
make build
make test-integration
```

## Implementation

### Commits

- `75e82f2` refactor: extract spawn logic into internal/core/spawn package

### Key Files Changed

**Created:**
- `internal/core/spawn/errors.go` - Error types (StateError, ConfigError, TmuxError, BinaryNotFoundError, PromptError)
- `internal/core/spawn/config.go` - MCP config generation (GenerateMCPConfig, WriteMCPConfig, RemoveMCPConfig)
- `internal/core/spawn/command.go` - Claude command building (BuildClaudeCommand, EscapePromptForShell, GenerateWindowName)
- `internal/core/spawn/state.go` - Session state detection (DetectTicketState, StateInfo with Normal/Active/Orphaned/Ended states)
- `internal/core/spawn/spawn.go` - Spawner struct with Spawn, Resume, Fresh methods
- `internal/core/spawn/spawn_test.go` - 17 unit tests covering all functionality

**Modified:**
- `internal/daemon/mcp/tools_architect.go` - Replaced ~150 lines of inline spawn logic with ~40 lines using the spawn package

### Important Decisions

- Used dependency injection via `StoreInterface` and `TmuxManagerInterface` for testability
- Added `TicketID` field to `SpawnResult` for consistency with output types
- State detection uses tmux `WindowExists` check to distinguish Active vs Orphaned sessions
- Package exposes type-checking helpers (`IsStateError`, `IsTmuxError`, etc.) for error handling

### Scope

Implemented as specified. No scope changes from original ticket.
