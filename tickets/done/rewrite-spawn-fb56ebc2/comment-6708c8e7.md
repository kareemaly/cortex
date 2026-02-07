---
id: 6708c8e7-615f-4552-928f-2e88cc09cd9e
author: claude
type: review_requested
created: 2026-02-07T09:31:27.212597Z
action:
    type: git_diff
    args:
        commit: 2e601e8
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Changes

Rewrites spawn orchestration to properly support the independent session store, fixing a real bug and adding comprehensive test coverage.

### Bug fix: `--resume resume` invalid flag
The orchestrate.go resume path set `SessionID: "resume"` as a placeholder, which flowed through to the launcher and generated `--resume resume` — telling Claude CLI to resume a conversation with ID "resume" (which doesn't exist). Fixed by:

1. **launcher.go**: Added `Resume bool` field to `LauncherParams`. Both `buildClaudeCommand` and `buildCopilotCommand` now emit bare `--resume` when `Resume: true` (resume most recent), or `--resume <id>` when `ResumeID` is set (specific conversation).
2. **spawn.go**: Removed `SessionID` empty-check validation from `Resume()`. Updated launcher params to set `Resume: req.SessionID == ""` and `ResumeID: req.SessionID`.
3. **orchestrate.go**: Removed `SessionID: "resume"` placeholder from the orphaned-resume path.

### Mock fidelity fix
Fixed `mockSessionStore.GetByTicketID` to return `storage.NotFoundError` for not-found sessions (matching the real store behavior) instead of `(nil, nil)`.

### New tests (13 Orchestrate tests + 1 launcher test)
- All 9 cells of the state/mode matrix (normal/active/orphaned × normal/resume/fresh)
- Backlog-to-progress auto-move after spawn
- Already-in-progress ticket does NOT move
- Default mode (empty string) behaves as "normal"
- Invalid mode returns ConfigError
- `TestWriteLauncherScript_BareResume` for the new bare `--resume` path

### Test updates
- `TestResume_Success` now tests bare `--resume` (no SessionID)
- Removed `TestResume_NoSessionID` (SessionID is now optional)

### Verification
- `make build` — passes
- `make lint` — 0 issues
- `make test` — all tests pass (33 spawn tests total)