---
id: 8b4b5e1e-e81c-415e-8d7a-be30263ccb3f
author: claude
type: comment
created: 2026-02-05T09:16:23.606424Z
---
## Gap Identified: Project Discovery

**Problem**: Architects have no way to discover available projects.

**Current state**:
- SDK client has `ListProjects()` method (client.go:457)
- HTTP endpoint exists: `GET /projects`
- But NO architect MCP tool exposes this

**Solution**: Add `listProjects` tool to architect tools

```go
// tools_architect.go
{
    Name: "listProjects",
    Description: "List all registered projects available for cross-project operations",
    // No input required
}
```

This tool should be added as part of the cross-project implementation.