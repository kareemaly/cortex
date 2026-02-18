---
id: 561cebb8-2aca-40b8-ba18-206dd20d7aeb
author: claude
type: done
created: 2026-02-12T14:47:17.389782Z
---
## Summary

Added prompt reset-to-default feature in the Config tab TUI, allowing users to revert ejected prompts back to built-in defaults.

### Changes (7 files, 200 additions)

**Backend:**
- `internal/daemon/api/types.go` — Added `ResetPromptRequest` struct
- `internal/daemon/api/prompts.go` — Added `Reset` handler (validates ejected file exists, `os.Remove()`, cleans empty parent dirs) and `removeEmptyParents` helper
- `internal/daemon/api/server.go` — Registered `POST /prompts/reset` route

**SDK:**
- `internal/cli/sdk/client.go` — Added `ResetPrompt(path string) error` method

**TUI:**
- `internal/cli/tui/config/keys.go` — Added `KeyX`, `KeyY`, `KeyN`, `KeyEscape` constants; updated help text with `[x] reset`
- `internal/cli/tui/config/model.go` — Added `showResetModal`/`resetPromptPath` state, `PromptResetMsg` type, `handleResetAction` (shows "Already using default" for non-ejected prompts), `handleResetModalKey` (y/n/esc), `resetPrompt` command, `renderResetModal`, and modal rendering in `View()`

**Docs:**
- `CLAUDE.md` — Added `/prompts/reset` to API endpoints list

### Verification
- `make build` — clean
- `make lint` — 0 issues
- `make test` — all pass
- Pre-push hooks passed
- Commit `6b34011` pushed to `origin/main`