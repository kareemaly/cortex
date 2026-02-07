---
id: 806ed58e-bae5-4c19-9066-3df0a42d4c50
author: claude
type: done
created: 2026-01-30T09:05:38.321067Z
---
## Summary

Implemented HTTP API endpoint for prompt resolution with extension fallback, addressing the requirement that all clients must go through the daemon HTTP API for prompt resolution.

## Changes Made

### 1. Enhanced Error Reporting (`internal/prompt/errors.go`)
- Restructured `NotFoundError` to include:
  - `Role`: "architect" or "ticket"
  - `TicketType`: for ticket prompts only
  - `Stage`: SYSTEM, KICKOFF, APPROVE
  - `SearchPaths`: all paths that were checked
- Error messages now display all searched locations for easier debugging

### 2. Added WithPath Methods (`internal/prompt/resolver.go`)
- Added `ResolvedPrompt` struct containing `Content` and `SourcePath`
- Added `ResolveArchitectPromptWithPath(stage string)` method
- Added `ResolveTicketPromptWithPath(ticketType, stage string)` method
- Original methods now delegate to WithPath versions for code reuse

### 3. Created HTTP API Endpoint (`internal/daemon/api/prompts.go`)
- New handler: `GET /prompts/resolve`
- Query parameters:
  - `role`: "architect" or "ticket" (required)
  - `stage`: "SYSTEM", "KICKOFF", or "APPROVE" (required)
  - `type`: ticket type name (required when role=ticket)
- Returns JSON: `{"content": "...", "source_path": "..."}`

### 4. Added Route (`internal/daemon/api/server.go`)
- Registered `/prompts/resolve` endpoint in project-scoped routes

### 5. Added SDK Client Method (`internal/cli/sdk/client.go`)
- Added `ResolvePromptRequest` and `ResolvePromptResponse` types
- Added `ResolvePrompt(req ResolvePromptRequest)` method

### 6. Added Tests (`internal/prompt/resolver_test.go`, `internal/prompt/prompt_test.go`)
- Tests for `ResolveArchitectPromptWithPath` verifying source path correctness
- Tests for `ResolveTicketPromptWithPath` verifying source path correctness
- Tests verifying error includes all searched paths
- Updated existing tests for new `NotFoundError` structure

## Files Modified
- `internal/prompt/errors.go`
- `internal/prompt/prompt.go`
- `internal/prompt/prompt_test.go`
- `internal/prompt/resolver.go`
- `internal/prompt/resolver_test.go`
- `internal/daemon/api/server.go`
- `internal/cli/sdk/client.go`

## Files Created
- `internal/daemon/api/prompts.go`

## Key Decisions

1. **WithPath methods as primary implementation**: The original `ResolveArchitectPrompt` and `ResolveTicketPrompt` methods now delegate to the new `*WithPath` versions, avoiding code duplication while maintaining backward compatibility.

2. **SearchPaths in error**: Instead of showing just one path, errors now show all paths that were searched, making it clear to users that both project and base paths were checked.

3. **Query parameters over request body**: Used query parameters for the GET endpoint since it's a read-only operation and keeps the API simple and RESTful.

## Verification
- All unit tests pass
- Linter shows 0 issues
- Build succeeds

## Commit
`5d383fb` - feat(prompt): add HTTP API for prompt resolution with extension fallback