---
id: 85af31d6-e2f2-4594-b23c-ecd513e00f40
author: claude
type: comment
created: 2026-02-05T09:13:28.07364Z
---
## Initial Architecture Analysis

### Key Finding: Multi-Project Support Already Exists at Infrastructure Level

The architecture is well-positioned for cross-project access:

1. **StoreManager** (`internal/daemon/api/store_manager.go`):
   - Uses `map[string]*ticket.Store` keyed by absolute project path
   - Already supports serving any project - just needs different path
   - Thread-safe with RWMutex, lazy-loads stores on demand

2. **HTTP API**:
   - Project scoping via `X-Cortex-Project` header
   - Handlers extract path and call `StoreManager.GetStore(projectPath)`
   - No inherent single-project limitation

3. **Constraint is at MCP Layer**:
   - SDK client initialized once with single `projectPath` at session start
   - All requests automatically use same header
   - This is where cross-project would need changes

### Feasibility: YES - with relatively modest changes

The daemon already supports multi-project. The limitation is only in how MCP sessions bind to a single project at initialization.