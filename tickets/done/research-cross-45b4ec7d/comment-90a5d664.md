---
id: 90a5d664-f62b-4fa8-9e02-333d828973f3
author: claude
type: comment
created: 2026-02-05T09:15:04.221078Z
---
## Research Conclusions

**Feasibility: YES** - Cross-project ticketing is feasible with modest changes.

### Key Decisions Made:
1. **Approach**: Option A - Explicit `project_path` parameter on architect tools
2. **Scope**: Full access (all tools except `deleteTicket`)
3. **Security**: Registry-based validation only (projects in `~/.cortex/settings.yaml`)

### Implementation Scope:
- ~4 files to modify
- SDK client: Add `WithProject()` method for per-request override
- Architect tools: Add optional `project_path` param to tool schemas
- Validation: Check target project is registered before cross-project ops

### Architecture Insight:
The daemon infrastructure (StoreManager, HTTP API, Event Bus) already supports multi-project access. The constraint is purely at the MCP session layer where SDK client binds to single project at init time.

Full details in plan file.