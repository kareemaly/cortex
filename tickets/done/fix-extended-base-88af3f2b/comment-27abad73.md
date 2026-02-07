---
id: 27abad73-8c98-4b94-abd3-13dc520d9ff8
author: claude
type: review_requested
created: 2026-01-30T09:13:27.996074Z
action:
    type: git_diff
    args:
        commit: 16732ed
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Summary

Fixed the extended base prompt path construction issue where the resolver was incorrectly adding `.cortex/` when constructing paths for extended base configs.

### Changes Made

1. **internal/prompt/prompt.go** - Added 3 new base-specific path functions:
   - `BasePromptsDir()` - Returns `{baseDir}/prompts` instead of `{baseDir}/.cortex/prompts`
   - `BaseArchitectPromptPath()` - Uses BasePromptsDir for architect prompts
   - `BaseTicketPromptPath()` - Uses BasePromptsDir for ticket prompts

2. **internal/prompt/resolver.go** - Updated 2 lines:
   - Line 55: Changed `ArchitectPromptPath(r.BaseRoot, stage)` to `BaseArchitectPromptPath(r.BaseRoot, stage)`
   - Line 102: Changed `TicketPromptPath(r.BaseRoot, ticketType, stage)` to `BaseTicketPromptPath(r.BaseRoot, ticketType, stage)`

3. **internal/prompt/resolver_test.go** - Updated tests:
   - Added `createBasePromptFile()` and `createBaseTicketPromptFile()` helpers that create prompts without `.cortex/` prefix
   - Updated all tests using `baseRoot` to use the new helpers
   - Fixed expected path assertion in "returns correct source path from base fallback" test

### Before/After
- **Before:** `~/.cortex/defaults/claude-code/.cortex/prompts/architect/SYSTEM.md`
- **After:** `~/.cortex/defaults/claude-code/prompts/architect/SYSTEM.md`

### Verification
- All unit tests pass: `go test ./internal/prompt/...`
- Full test suite passes: `make test`
- Linter passes: `make lint`