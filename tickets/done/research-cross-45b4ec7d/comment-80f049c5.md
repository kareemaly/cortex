---
id: 80f049c5-c898-46ff-80c2-87e6f06fd52d
author: claude
type: comment
created: 2026-02-05T09:14:06.584867Z
---
## Design Options Summary

**Recommended: Option A - Explicit Project Parameter**

Add optional `project_path` parameter to architect tools. Minimal change, backward compatible, clear intent.

**Files to modify:**
- `internal/cli/sdk/client.go` - Add `WithProject()` method
- `internal/daemon/mcp/tools_architect.go` - Add param to tool schemas
- `internal/daemon/api/projects.go` - Add validation endpoint

**Security:** Only allow access to registered projects (validate against `~/.cortex/settings.yaml`).

See full plan file for implementation details.