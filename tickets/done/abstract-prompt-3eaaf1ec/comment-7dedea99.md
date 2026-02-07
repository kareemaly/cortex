---
id: 7dedea99-07c4-4551-b316-f6aaaced5050
author: claude
type: review_requested
created: 2026-01-30T09:04:57.310503Z
action:
    type: git_diff
    args:
        commit: 5d383fb
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Summary

Implemented HTTP API endpoint for prompt resolution with extension fallback. The implementation includes:

### Changes Made

1. **Enhanced `NotFoundError`** (`internal/prompt/errors.go`):
   - Added `Role`, `TicketType`, `Stage`, and `SearchPaths` fields
   - Error message now lists all searched paths for better debugging

2. **Added `*WithPath` methods** (`internal/prompt/resolver.go`):
   - `ResolveArchitectPromptWithPath(stage string)` - returns `ResolvedPrompt` with content and source path
   - `ResolveTicketPromptWithPath(ticketType, stage string)` - returns `ResolvedPrompt` with content and source path
   - Original methods now delegate to the WithPath versions

3. **New HTTP endpoint** (`internal/daemon/api/prompts.go`):
   - `GET /prompts/resolve` - resolves prompts with extension fallback
   - Query parameters: `role` (architect/ticket), `stage` (SYSTEM/KICKOFF/APPROVE), `type` (for ticket prompts)
   - Returns JSON with `content` and `source_path`

4. **SDK client method** (`internal/cli/sdk/client.go`):
   - Added `ResolvePrompt(req ResolvePromptRequest)` method for clients to use

5. **Tests**:
   - Added tests for `WithPath` methods verifying correct source path return
   - Added tests for error path reporting (verifies all searched paths are included)
   - Updated existing tests to use new `NotFoundError` structure

### Acceptance Criteria Met

- ✅ Prompt resolution respects `extend` fallback chain (already working, now exposed via API)
- ✅ All daemon prompt loading uses the abstracted resolver (was already using it)
- ✅ TUIs and CLI can now go through daemon API for prompt resolution (new endpoint)
- ✅ Clear error message when prompt not found shows all searched locations