---
id: 567b4f0d-d11f-478a-8cfa-6b29a32a91fa
author: claude
type: ticket_done
created: 2026-01-26T16:29:22.580722Z
---
## Summary

Extracted a shared `Orchestrate` function that serves as the single source of truth for spawning ticket agent sessions. Both the HTTP API handler and MCP tool now delegate to this function, eliminating duplicated and divergent spawn orchestration logic.

## Files Changed

### New
- `internal/core/spawn/orchestrate.go` — Shared orchestration function with types (`OrchestrateStore`, `OrchestrateRequest`, `OrchestrateDeps`, `OrchestrateResult`, `Outcome`)

### Modified
- `internal/daemon/api/tickets.go` — Spawn handler reduced from ~185 lines to ~50 lines; removed `projectconfig` and `filepath` imports
- `internal/daemon/mcp/tools_architect.go` — handleSpawnSession reduced from ~170 lines to ~50 lines; added `errors` import for `errors.As`; added mode pre-validation
- `internal/daemon/mcp/tools_test.go` — Updated 3 tests to match unified behavior

## Key Decisions

1. **Orchestrate returns `OutcomeAlreadyActive` instead of error for normal+active** — Lets callers handle differently (HTTP focuses window + 200; MCP returns StateConflictError)

2. **Soft spawn failures converted to errors** — When spawner returns `Success=false` (prompt load failure, tmux error), Orchestrate converts to `fmt.Errorf` so callers only handle the error path

3. **Mode pre-validation in MCP handler** — Added before Orchestrate call to preserve the existing MCP contract of returning `VALIDATION_ERROR` for invalid modes (Orchestrate returns `ConfigError` which maps differently)

4. **Test updates for unified behavior** — `TestHandleSpawnSessionNoAutoMove` renamed to `TestHandleSpawnSessionAutoMovesToProgress` because MCP now auto-moves backlog→progress (matching HTTP behavior, as intended by the ticket)

## Divergences Resolved

| Aspect | Before (HTTP) | Before (MCP) | After (shared) |
|--------|--------------|--------------|----------------|
| UseWorktree | Always false | From project config | From project config |
| CortexdPath | Not passed | From config | Passed through deps |
| Fresh mode | Manual EndSession + Spawn | spawner.Fresh() | spawner.Fresh() |
| State matrix | Partial | Complete | Complete |
| Post-spawn move | In handler | Not done | In Orchestrate |
| Agent resolution | From project config | Input or "claude" | Input > config > "claude" |

## Verification

- `make build` ✅
- `make lint` ✅ (0 issues)
- `make test` ✅ (all unit tests pass)
- Integration test failures are pre-existing (verified on clean main)

## Commit

`0f1f3d8` — refactor: extract shared Orchestrate function for spawn logic