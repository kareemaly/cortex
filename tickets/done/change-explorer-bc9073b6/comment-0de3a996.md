---
id: 0de3a996-c81f-4018-a6c7-0826885ad57c
author: claude
type: done
created: 2026-02-12T15:03:48.432298Z
---
## Summary

Replaced the subtle `▎` vertical bar selection indicator in docs and config explorer panes with accent-colored bold text styling for better visibility.

### Changes (4 files)
- **`internal/cli/tui/config/styles.go`** — `selectedStyle` now uses accent color (purple/62) + bold; added `unfocusedSelectedStyle` (gray/245); removed `selectedIndicator`
- **`internal/cli/tui/docs/styles.go`** — same style changes
- **`internal/cli/tui/config/model.go`** — replaced `▎` bar with space; uses named styles for focused/unfocused selection
- **`internal/cli/tui/docs/model.go`** — same rendering changes

### Verification
- Build, lint, and all unit tests pass
- Pushed to origin/main (commit f0df444)