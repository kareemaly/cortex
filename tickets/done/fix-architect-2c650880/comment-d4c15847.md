---
id: d4c15847-101d-41ad-ba2b-1c28bb6aef7b
author: claude
type: done
created: 2026-02-10T09:59:06.020094Z
---
## Root Cause

In `internal/core/spawn/spawn.go`, both `Spawn()` and `Resume()` used a fixed identifier `"architect"` for all architect sessions regardless of project. This identifier determines temp file paths (MCP config, prompt, system prompt, launcher script, settings). When meta spawned architects for two different projects, the second architect's temp files overwrote the first's before the first's async launcher script could read them, causing wrong project tickets/docs to be injected.

## Resolution

Changed the architect identifier from `"architect"` to `"architect-" + req.TmuxSession` in both `Spawn()` (line 250) and `Resume()` (line 378). `req.TmuxSession` is unique per project and validated to be filename-safe by `validateTmuxName()`.

## Files Changed

- `internal/core/spawn/spawn.go` — 2 lines changed (one in `Spawn()`, one in `Resume()`)

## Verification

- `make test` — all unit tests pass
- `make lint` — 0 issues
- Merged to main