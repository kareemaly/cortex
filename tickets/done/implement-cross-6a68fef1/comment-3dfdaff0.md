---
id: 3dfdaff0-d369-4bfd-8662-80c9270df7de
author: claude
type: done
created: 2026-02-05T09:46:43.033844Z
---
## Summary

Implemented cross-project ticketing support for architects, enabling them to access, create, and spawn tickets across different registered projects from a single MCP session.

## Changes Made

### SDK Client (`internal/cli/sdk/client.go`)
- Added `WithProject(projectPath string) *Client` method that returns a new client targeting a different project while sharing the underlying HTTP client

### MCP Types (`internal/daemon/mcp/types.go`)
- Added `ListProjectsInput` (empty struct, no params)
- Added `ProjectSummary` struct with Path, Title, Exists fields
- Added `ListProjectsOutput` with Projects slice
- Added optional `ProjectPath` field to 9 input structs:
  - ListTicketsInput, ReadTicketInput, CreateTicketInput, UpdateTicketInput
  - MoveTicketInput, SpawnSessionInput, ArchitectAddCommentInput
  - UpdateDueDateInput, ClearDueDateInput

### MCP Tools (`internal/daemon/mcp/tools_architect.go`)
- Added `validateProjectPath()` helper - validates project is registered and directory exists
- Added `getClientForProject()` helper - returns SDK client for target project or default
- Added `listProjects` tool registration and handler
- Updated 9 handlers to validate project path and use appropriate client

### Documentation (`CLAUDE.md`)
- Added `listProjects` tool to MCP tools table
- Added cross-project support note explaining the feature and exceptions

## Design Decisions
- `deleteTicket` excluded from cross-project support (too risky)
- `getCortexConfigDocs` excluded (project-specific configuration)
- Project validation returns clear error messages with registration hints

## Verification
- Build: passed
- Unit tests: all passed
- Lint: 0 issues
- Push: successful (commit 74faa72)