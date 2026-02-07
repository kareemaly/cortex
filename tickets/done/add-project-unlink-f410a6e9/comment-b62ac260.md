---
id: b62ac260-8828-48a1-9b04-3c2d35f2b438
author: claude
type: review_requested
created: 2026-02-05T11:40:15.095004Z
action:
    type: git_diff
    args:
        commit: fa6e1ef
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/68e84076-4ec4-420d-9645-30af191f6d29
---
## Summary

Added project unlink capability to the dashboard TUI, allowing users to remove projects from the global registry without deleting any files.

## Changes

### API Layer (`internal/daemon/api/`)
- **projects.go**: Added `UnlinkProjectHandler` that handles `DELETE /projects?path=<path>` requests
- **server.go**: Added the `DELETE /projects` route in the global endpoints section

### SDK Layer (`internal/cli/sdk/client.go`)
- Added `UnlinkProject(projectPath string) error` method that calls the new API endpoint

### Dashboard TUI (`internal/cli/tui/dashboard/`)
- **keys.go**: Added key constants (`KeyUnlink`, `KeyYes`, `KeyNo`, `KeyEscape`) and updated help text to include `[u]nlink`
- **model.go**:
  - Added `UnlinkProjectMsg` message type
  - Added `showUnlinkConfirm` and `unlinkProjectPath` fields for confirmation state
  - Added confirmation mode handling in `handleKeyMsg()` (y/n/esc)
  - Added `handleUnlinkProject()` to initiate unlink confirmation
  - Added `unlinkProject()` command to execute the API call
  - Added confirmation dialog rendering in `View()`

## UX Flow

1. User navigates to a project row with j/k
2. User presses `u` to initiate unlink
3. Confirmation dialog appears: "Unlink project 'name'? [y]es [n]o"
4. User confirms with `y` or cancels with `n`/`esc`
5. On success: project disappears from list, "Project unlinked" status shown
6. On cancel: "Unlink cancelled" status shown

## Testing

- `make build` - passes
- `make test` - all unit tests pass
- `make lint` - no issues