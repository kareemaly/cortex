---
id: 439b9c78-1959-4c9c-aee6-9d8565d23565
author: claude
type: review_requested
created: 2026-02-10T09:58:39.965375Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/877a28b7-f308-4268-b1ce-b976fac71ae3
        commit: fe76c96
---
## Root Cause

In `internal/core/spawn/spawn.go`, both `Spawn()` (line 250) and `Resume()` (line 378) used a fixed identifier `"architect"` for all architect sessions regardless of project. This identifier determines temp file paths for MCP config, prompt, system prompt, launcher script, and settings. When two architects are spawned for different projects, the second overwrites the first's files before the first's async launcher script reads them, causing the wrong project's tickets/docs to be injected.

## Fix

Changed the architect identifier from `"architect"` to `"architect-" + req.TmuxSession` in both `Spawn()` and `Resume()`. This makes each architect's temp files project-scoped since `req.TmuxSession` is unique per project and already validated to be filename-safe by `validateTmuxName()`.

## Verification

- `make test` — all unit tests pass
- `make lint` — 0 issues
- Single file changed, 2 lines modified (minimal, targeted fix)