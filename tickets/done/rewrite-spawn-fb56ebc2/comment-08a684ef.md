---
id: 08a684ef-705a-4dd0-ae52-bb2a4659deef
author: claude
type: done
created: 2026-02-07T09:32:02.062738Z
---
## Summary

Rewrote spawn orchestration to properly support the independent session store, fixing a real bug and adding comprehensive test coverage.

### Changes Made

**Files modified (4):**
- `internal/core/spawn/launcher.go` — Added `Resume bool` field to `LauncherParams`. Both `buildClaudeCommand` and `buildCopilotCommand` now emit bare `--resume` when `Resume: true`, or `--resume <id>` when `ResumeID` is set.
- `internal/core/spawn/spawn.go` — Removed `SessionID` empty-check validation from `Resume()`. Updated launcher params to set `Resume: req.SessionID == ""` and `ResumeID: req.SessionID`.
- `internal/core/spawn/orchestrate.go` — Removed `SessionID: "resume"` placeholder from the orphaned-resume path.
- `internal/core/spawn/spawn_test.go` — Fixed mock `GetByTicketID` to return `NotFoundError` (matching real store). Updated `TestResume_Success` for bare `--resume`. Removed `TestResume_NoSessionID`. Added `TestWriteLauncherScript_BareResume`. Added 13 `Orchestrate()` tests covering all 9 state/mode matrix cells, backlog-to-progress move, default mode, and invalid mode.

### Key Decisions
- Used a `Resume bool` field alongside `ResumeID string` in `LauncherParams` to cleanly separate bare `--resume` (most recent conversation) from `--resume <id>` (specific conversation).
- Made `SessionID` optional in `ResumeRequest` rather than required — empty means bare resume.

### Verification
- `make build` — passes
- `make lint` — 0 issues  
- `make test` — all tests pass (33 spawn tests total)

### Commit
`2e601e8` on `feat/frontmatter-storage` branch