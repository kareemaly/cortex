---
id: b967e2bf-98be-4505-8361-540e5b974c99
author: claude
type: review_requested
created: 2026-02-05T09:46:12.205899Z
action:
    type: git_diff
    args:
        commit: 74faa72
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Summary

Implemented cross-project ticketing support for architects, enabling them to access, create, and spawn tickets across different registered projects from a single MCP session.

## Changes

### SDK Client (`internal/cli/sdk/client.go`)
- Added `WithProject()` method that returns a new client targeting a different project while sharing the underlying HTTP client for efficiency

### MCP Types (`internal/daemon/mcp/types.go`)
- Added `ListProjectsInput` and `ListProjectsOutput` types for the new listProjects tool
- Added `ProjectSummary` struct for project information
- Added optional `ProjectPath` field to 9 input structs:
  - `ListTicketsInput`
  - `ReadTicketInput`
  - `CreateTicketInput`
  - `UpdateTicketInput`
  - `MoveTicketInput`
  - `SpawnSessionInput`
  - `ArchitectAddCommentInput`
  - `UpdateDueDateInput`
  - `ClearDueDateInput`
- Did NOT add to `DeleteTicketInput` (too risky for cross-project)
- Did NOT add to `GetCortexConfigDocsInput` (project-specific)

### MCP Tools (`internal/daemon/mcp/tools_architect.go`)
- Added `validateProjectPath()` helper that validates project is registered and exists
- Added `getClientForProject()` helper that returns SDK client for target project
- Added `listProjects` tool registration and handler
- Updated all 9 handlers to:
  1. Validate project path first
  2. Get client for target project
  3. Use client instead of s.sdkClient

### Documentation (`CLAUDE.md`)
- Added `listProjects` to MCP tools table
- Added note about cross-project support and exceptions

## Verification

- `make build` - passed
- `make test` - all tests passed
- `make lint` - 0 issues