---
id: 34bef9a5-14d9-497c-8b73-55746b96ea01
author: claude
type: done
created: 2026-02-05T09:31:29.837716Z
---
## Research Complete: Cross-Project Ticketing for Architects

### Key Findings

1. **Feasibility: YES** - Cross-project access is feasible with modest architectural changes.

2. **Architecture Insight**: The daemon infrastructure (StoreManager, HTTP API, Event Bus) already supports multi-project access. The limitation is purely at the MCP session layer where the SDK client binds to a single project at initialization.

3. **Current Constraint**: SDK client initialized once with `projectPath`, all requests use same `X-Cortex-Project` header.

### Recommended Approach

**Option A: Explicit Project Parameter**
- Add optional `project_path` parameter to all architect tools
- If omitted → uses session's default project (backward compatible)
- If provided → uses specified project (cross-project operation)

### Implementation Scope

**Files to modify:**
- `internal/cli/sdk/client.go` - Add `WithProject(path)` method for per-request override
- `internal/daemon/mcp/tools_architect.go` - Add `project_path` param to tool schemas
- `internal/daemon/mcp/tools_architect.go` - Add new `listProjects` tool for discovery

**New tool required:**
- `listProjects` - Allows architects to discover available projects (SDK method already exists)

**Tool scope:**
- Full access (all tools except `deleteTicket`)
- Security: Registry-based validation (only projects in `~/.cortex/settings.yaml`)

### Use Cases Enabled
1. Architect in `frontend` creates/spawns tickets in `api` project
2. View/manage tickets across monorepo packages
3. Cross-codebase visibility and coordination

### Artifacts
- Plan file: `/Users/kareemelbahrawy/.claude/plans/humming-roaming-hedgehog.md`
- Ticket comments documenting analysis progression

### Next Steps (if implementing)
1. Add `WithProject()` method to SDK client
2. Add `listProjects` architect tool
3. Add optional `project_path` param to all architect tool schemas
4. Add validation against project registry
5. Add integration tests for cross-project scenarios