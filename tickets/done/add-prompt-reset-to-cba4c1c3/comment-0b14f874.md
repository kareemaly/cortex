---
id: 0b14f874-c0a7-4cbd-984b-7363ada3377b
author: claude
type: review_requested
created: 2026-02-12T14:45:07.731655Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Add prompt reset-to-default in Config tab TUI

### Changes

**API endpoint — `POST /prompts/reset`** (`internal/daemon/api/prompts.go`):
- New `Reset` handler on `PromptHandlers` that accepts `{ "path": "architect/SYSTEM.md" }`, verifies the ejected file exists, removes it with `os.Remove()`, and cleans up empty parent directories up to `.cortex/prompts` root
- Added `removeEmptyParents()` helper
- Returns `ExecuteActionResponse{ Success: true, Message: "Prompt reset to default" }`

**Route registration** (`internal/daemon/api/server.go`):
- Added `r.Post("/reset", promptHandlers.Reset)` in the `/prompts` route group

**Request type** (`internal/daemon/api/types.go`):
- Added `ResetPromptRequest` struct with `Path string`

**SDK client** (`internal/cli/sdk/client.go`):
- Added `ResetPrompt(path string) error` method — POST to `/prompts/reset`

**TUI Config tab** (`internal/cli/tui/config/keys.go`, `internal/cli/tui/config/model.go`):
- Added key constants: `KeyX`, `KeyY`, `KeyN`, `KeyEscape`
- Updated help text to include `[x] reset`
- Added `showResetModal` / `resetPromptPath` model fields
- Added `PromptResetMsg` message type with handler (reload on success, show error on failure)
- `x` on a default prompt shows "Already using default"; on an ejected prompt shows y/n confirmation modal
- `handleResetModalKey()` — `y` triggers reset, `n`/`esc` dismisses
- `renderResetModal()` renders status + help bar inline (same pattern as kanban delete modal)
- `View()` updated to show reset modal in place of status/help bar when active

### Verification
- `make build` — compiles cleanly
- `make lint` — 0 issues
- `make test` — all tests pass