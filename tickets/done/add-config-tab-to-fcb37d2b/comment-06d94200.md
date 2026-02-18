---
id: 06d94200-4019-483f-ad9b-73bc0e73018b
author: claude
type: review_requested
created: 2026-02-12T12:08:50.25074Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Config Tab Implementation for Architect Companion TUI

### Summary
Added a third "Config" tab to the `cortex kanban` TUI that allows browsing, ejecting, and editing project prompts and configuration directly from the TUI.

### Changes by file

**Response types (`internal/types/response.go`)**
- Added `PromptFileInfo`, `PromptGroupInfo`, `ListPromptsResponse` types

**API types (`internal/daemon/api/types.go`)**
- Added re-exports for the 3 new response types
- Added `EjectPromptRequest` and `EditPromptRequest` request types

**Prompt handlers (`internal/daemon/api/prompts.go`)**
- Added `List` handler (`GET /prompts`) - walks base prompts directory, detects ejection status, groups files by role/subgroup
- Added `Eject` handler (`POST /prompts/eject`) - copies prompt from base to project for customization
- Added `Edit` handler (`POST /prompts/edit`) - opens ejected prompt in $EDITOR via tmux popup

**Config handlers (`internal/daemon/api/config_handlers.go`)**
- Added `EditProjectConfig` handler (`POST /config/project/edit`) - opens cortex.yaml in $EDITOR via tmux popup

**Routes (`internal/daemon/api/server.go`)**
- Converted prompt routes to `r.Route("/prompts", ...)` sub-router with resolve, list, eject, edit
- Added `POST /config/project/edit` route

**SDK client (`internal/cli/sdk/client.go`)**
- Added re-exports for `PromptFileInfo`, `PromptGroupInfo`, `ListPromptsResponse`
- Added `ListPrompts()`, `EjectPrompt()`, `EditPromptInEditor()`, `EditProjectConfigInEditor()` methods

**New TUI config package (`internal/cli/tui/config/`)**
- `keys.go` - Key constants and help text
- `styles.go` - Lipgloss styles matching existing docs palette
- `model.go` - Full Bubbletea model with:
  - Two-pane layout (30% list / 70% preview)
  - cortex.yaml at top, then grouped prompt files with section headers
  - Ejection status badges (○ default / ● ejected)
  - `e` key: eject (default) or edit (ejected) prompts, edit config
  - `c` key: shortcut to edit cortex.yaml
  - `h/l` pane switching, `j/k` navigation that skips section headers
  - Glamour markdown rendering for preview
  - Log viewer toggle with `!`

**Views wrapper (`internal/cli/tui/views/model.go`)**
- Added `viewConfig` to view enum
- Added `config.Model` field
- Wired init, update (window size, keys, non-key msgs), view, and tab bar

### Verification
- `make build` - compiles cleanly
- `make lint` - 0 issues
- `make test` - all unit tests pass