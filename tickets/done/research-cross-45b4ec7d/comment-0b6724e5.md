---
id: 0b6724e5-cf8c-46c9-883e-803370833b35
author: claude
type: review_requested
created: 2026-02-05T09:15:15.141464Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Research Complete: Cross-Project Ticketing for Architects

### Research Question
Is it feasible for architects to access, create, and spawn tickets across different projects from a single session?

### Answer: YES - Feasible with modest changes

---

### Key Finding
The daemon infrastructure **already supports multi-project access**:
- `StoreManager` uses `map[string]*ticket.Store` keyed by project path
- HTTP API accepts any `X-Cortex-Project` header value
- Event bus is keyed by project for proper isolation

The constraint is **only at the MCP layer** where SDK client binds to a single project at initialization.

---

### Recommended Approach: Explicit Project Parameter

Add optional `project_path` parameter to architect tools. If provided, override session default.

**Files to modify:**
1. `internal/cli/sdk/client.go` - Add `WithProject(path)` method
2. `internal/daemon/mcp/tools_architect.go` - Add `project_path` param to schemas
3. `internal/daemon/api/projects.go` - Add validation helper (optional)

**Scope decisions:**
- Full tool access (all except `deleteTicket`)
- Security: Registry-based validation (only projects in `~/.cortex/settings.yaml`)

---

### Use Cases Enabled
1. Architect in `frontend` creates ticket in `api` project
2. View/manage tickets across monorepo packages
3. Spawn work in dependency projects from current session

---

### Deliverables
- [x] Summary of architectural implications
- [x] Recommended approach (Option A - explicit project param)
- [x] List of files/components to modify
- [x] Security model (registry validation)
- [x] Full plan file at `/Users/kareemelbahrawy/.claude/plans/humming-roaming-hedgehog.md`