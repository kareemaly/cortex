---
id: 3f858e6d-1ce5-4dba-a77c-30b8d5c4450f
author: claude
type: done
created: 2026-02-05T12:37:04.823705Z
---
## Summary

Implemented project unlink capability for the dashboard TUI, allowing users to remove projects from the global registry (~/.cortex/settings.yaml) without deleting any files.

## Changes Made

### Files Modified (5 files, +151 lines)

1. **internal/daemon/api/projects.go** - Added `UnlinkProjectHandler` for `DELETE /projects?path=<path>` endpoint
2. **internal/daemon/api/server.go** - Added route for the new endpoint
3. **internal/cli/sdk/client.go** - Added `UnlinkProject(projectPath string) error` SDK method
4. **internal/cli/tui/dashboard/keys.go** - Added key constants (u, y, n, esc) and updated help text
5. **internal/cli/tui/dashboard/model.go** - Added confirmation state, handlers, message type, and view rendering

## Feature Details

- Press `u` on a project row to initiate unlink
- Confirmation dialog shows project name and path
- Confirm with `y`, cancel with `n` or `esc`
- Works on both active and stale projects
- Only available on project rows, not session rows

## Verification

- Build: passes
- Unit tests: all pass
- Lint: no issues
- Merged to main and pushed to origin