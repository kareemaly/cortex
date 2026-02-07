---
id: 477574c2-025c-454c-8640-7b54ab0191b1
author: claude
type: ticket_done
created: 2026-01-27T11:29:58.700661Z
---
## Summary

Implemented a global project registry in `~/.cortex/settings.yaml` that tracks all registered Cortex projects. This enables cross-project visibility features like the daemon dashboard TUI.

## Changes Made

### 1. Daemon Config (`internal/daemon/config/config.go`) — Modified
- Added `ProjectEntry` struct with `Path` and `Title` fields (yaml-tagged)
- Added `Projects []ProjectEntry` field to `Config` struct
- Extracted `configPath()` helper for reuse across Load/Save
- Added `LoadFromFile(path)` to enable testing with arbitrary file paths
- Added `Save()` and `SaveToFile(path)` methods on `*Config`
- Added `RegisterProject(absPath, title) bool` — idempotent, returns true if newly added
- Added `UnregisterProject(absPath) bool` — returns true if found and removed

### 2. Config Tests (`internal/daemon/config/config_test.go`) — New
- 7 unit tests: register, idempotent register, unregister, not-found unregister, save/load round-trip, missing file load, file creation

### 3. Auto-Registration (`internal/install/install.go`, `internal/install/result.go`) — Modified
- Added `registerProject()` helper that loads global config, registers the project, and saves
- Called in `Run()` after `setupProject()` succeeds (non-fatal: errors stored on result, don't fail init)
- Added `Registered bool` and `RegistrationError error` fields to `Result`

### 4. Init Command (`cmd/cortex/commands/init.go`) — Modified
- Prints "Global registry:" section showing registration status (checkmark/bullet/cross)

### 5. API Endpoint (`internal/daemon/api/projects.go`) — New
- `GET /projects` handler loads global config, iterates entries, checks `.cortex/` existence, gets ticket counts via StoreManager (best-effort)
- Response types: `ProjectResponse` (Path, Title, Exists, Counts), `ProjectTicketCounts` (Backlog, Progress, Review, Done)

### 6. Server Route (`internal/daemon/api/server.go`) — Modified
- Registered `GET /projects` route alongside `/health` (outside project-scoped group)

### 7. SDK Client (`internal/cli/sdk/client.go`) — Modified
- Added `ProjectTicketCounts`, `ProjectResponse`, `ListProjectsResponse` types
- Added `ListProjects()` method (no project header needed, like Health)

### 8. Projects Command (`cmd/cortex/commands/projects.go`) — New
- `cortex projects` with `--json` flag
- Table output: TITLE, PATH, BACKLOG, PROGRESS, REVIEW, DONE, STATUS columns
- Stale projects show "stale" status and "-" for counts
- Empty state message pointing to `cortex init`

### 9. Register/Unregister Commands (`cmd/cortex/commands/register.go`) — New
- `cortex register [path]` — validates `.cortex/` exists, registers in global config
- `cortex unregister [path]` — removes from global config
- Both default to cwd, operate directly on config file (no daemon needed)

## Key Decisions

- **Non-destructive stale handling**: Stale entries stay in settings.yaml, marked with `exists: false` in API response and "stale" in CLI. Users explicitly remove via `cortex unregister`.
- **Non-fatal auto-registration**: If registration fails during `cortex init`, the error is reported but init doesn't fail.
- **No project header for /projects**: The endpoint is global (like /health), not scoped to a single project.
- **Best-effort ticket counts**: If store creation fails for a project, counts are omitted rather than erroring the whole response.

## Files Changed (10)
- `internal/daemon/config/config.go` — Modified
- `internal/daemon/config/config_test.go` — New
- `internal/install/install.go` — Modified
- `internal/install/result.go` — Modified
- `cmd/cortex/commands/init.go` — Modified
- `internal/daemon/api/projects.go` — New
- `internal/daemon/api/server.go` — Modified
- `internal/cli/sdk/client.go` — Modified
- `cmd/cortex/commands/projects.go` — New
- `cmd/cortex/commands/register.go` — New

## Verification
- `make build` — compiles successfully
- `make lint` — 0 issues
- `make test` — all tests pass (including 7 new config tests)