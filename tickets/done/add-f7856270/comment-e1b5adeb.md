---
id: e1b5adeb-e1c6-403d-936f-d031183dbe00
author: claude
type: done
created: 2026-02-03T08:53:50.173419Z
---
## Summary

Implemented the `getCortexConfigDocs` MCP tool for architect sessions that returns embedded CONFIG_DOCS.md content for the project's configured agent type.

## Changes Made

### Files Modified
1. **internal/install/embed.go**
   - Added `GetConfigDocs(configName string)` function to read embedded CONFIG_DOCS.md files from the defaults directory

2. **internal/daemon/mcp/types.go**
   - Added `GetCortexConfigDocsInput` struct (empty, as the tool takes no parameters)
   - Added `GetCortexConfigDocsOutput` struct with `Content` and `ConfigName` fields

3. **internal/daemon/mcp/tools_architect.go**
   - Registered `getCortexConfigDocs` tool in `registerArchitectTools()`
   - Implemented `handleGetCortexConfigDocs` handler with logic to:
     - Determine config name from architect agent type (e.g., "claude" → "claude-code")
     - First check resolved extend path for user customization
     - Fall back to embedded file if not found in extend path

## Verification
- Build: `make build` - passed
- Lint: `make lint` - passed (0 issues)
- Tests: `make test` - all unit tests passed

## Commit
- Hash: 8b9f088
- Message: "feat(mcp): add getCortexConfigDocs tool for architect sessions"

Merged to main and pushed to origin.