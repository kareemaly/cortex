---
id: be880147-5dfd-4098-bd01-cd6cdad35737
author: claude
type: done
created: 2026-01-30T09:15:23.140899Z
---
## Summary

Fixed the extended base prompt path construction issue where the prompt resolver was incorrectly adding a `.cortex/` segment when constructing paths for extended base configurations.

### Problem

The `extend` attribute in project configs points to a **cortex config directory** (equivalent to `.cortex/`), but the code was treating it as a project root and adding an extra `.cortex/` segment:

- **Broken path:** `~/.cortex/defaults/claude-code/.cortex/prompts/architect/SYSTEM.md`
- **Correct path:** `~/.cortex/defaults/claude-code/prompts/architect/SYSTEM.md`

### Root Cause

The `PromptsDir()` function was being used for both project roots and extended bases, but these have different structures:
- Project roots: `{projectRoot}/.cortex/prompts/`
- Extended bases (cortex config dirs): `{baseDir}/prompts/`

### Solution

Added separate path functions for base config directories that don't include the `.cortex/` prefix:

1. `BasePromptsDir(cortexConfigDir)` - Returns `{baseDir}/prompts`
2. `BaseArchitectPromptPath(cortexConfigDir, stage)` - For architect prompts
3. `BaseTicketPromptPath(cortexConfigDir, ticketType, stage)` - For ticket prompts

Updated the resolver to use these functions when looking up prompts from the extended base.

### Files Modified

1. **internal/prompt/prompt.go** (+17 lines)
   - Added `BasePromptsDir()`, `BaseArchitectPromptPath()`, `BaseTicketPromptPath()` functions

2. **internal/prompt/resolver.go** (+2/-2 lines)
   - Line 55: `ArchitectPromptPath` → `BaseArchitectPromptPath` for base fallback
   - Line 102: `TicketPromptPath` → `BaseTicketPromptPath` for base fallback

3. **internal/prompt/resolver_test.go** (+38/-8 lines)
   - Added `createBasePromptFile()` and `createBaseTicketPromptFile()` test helpers
   - Updated 8 tests to use the new helpers for base configs
   - Fixed expected path assertion in "returns correct source path from base fallback"

### Key Decisions

- **Separate functions vs. parameter flag:** Chose to add new `Base*` functions rather than adding a boolean parameter to existing functions. This keeps the API clear and self-documenting - callers explicitly choose between project paths and base paths.

- **Test helper approach:** Added separate `createBase*` helpers rather than adding a parameter to existing helpers. This mirrors the production code structure and makes test intent clearer.

### Verification

- All unit tests pass: `go test ./internal/prompt/...`
- Full test suite passes: `make test`
- Linter passes: `make lint` (0 issues)

### Commit

`16732ed` - fix(prompt): use correct path structure for extended base configs