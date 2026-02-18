---
id: ce7970e4-ee64-4e20-b72a-23cbe4c1dfb6
author: claude
type: done
created: 2026-02-12T12:11:34.388345Z
---
## Summary

Implemented a Config tab for the `cortex kanban` TUI, allowing users to browse, eject, and edit project prompts and configuration directly from the companion interface.

### What was done

**Backend (API + SDK):**
- Added 3 new response types: `PromptFileInfo`, `PromptGroupInfo`, `ListPromptsResponse`
- Added 4 new API endpoints: `GET /prompts` (list all prompts with ejection status), `POST /prompts/eject` (copy prompt from base to project), `POST /prompts/edit` (open ejected prompt in $EDITOR via tmux popup), `POST /config/project/edit` (open cortex.yaml in $EDITOR via tmux popup)
- Added 4 new SDK client methods: `ListPrompts()`, `EjectPrompt()`, `EditPromptInEditor()`, `EditProjectConfigInEditor()`

**Frontend (TUI):**
- Created new `internal/cli/tui/config/` package with keys, styles, and full Bubbletea model
- Two-pane layout (30% list / 70% preview) with cortex.yaml at top and grouped prompt files with section headers
- Ejection status badges (○ default / ● ejected)
- Actions: `e` to eject/edit, `c` for config shortcut, `h/l` pane switching, `j/k` navigation skipping headers
- Glamour markdown rendering for preview pane
- Wired into views wrapper as third tab (Kanban | Docs | Config)

**Documentation:**
- Updated CLAUDE.md: TUI components path and API endpoints description

### Files changed
- `internal/types/response.go` - 3 new types
- `internal/daemon/api/types.go` - re-exports + request types
- `internal/daemon/api/prompts.go` - List, Eject, Edit handlers
- `internal/daemon/api/config_handlers.go` - EditProjectConfig handler
- `internal/daemon/api/server.go` - 4 new routes
- `internal/cli/sdk/client.go` - 4 new methods + re-exports
- `internal/cli/tui/config/keys.go` - **new**
- `internal/cli/tui/config/styles.go` - **new**
- `internal/cli/tui/config/model.go` - **new**
- `internal/cli/tui/views/model.go` - Config tab wiring
- `CLAUDE.md` - docs update

### Verification
- `make build` ✓
- `make lint` ✓ (0 issues)
- `make test` ✓ (all passing)
- Pre-push hook ✓ (lint + build)