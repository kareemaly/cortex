---
id: 1813e9d3-cc7e-44e3-ad2f-032f673bfa31
author: claude
type: review_requested
created: 2026-02-12T15:02:04.020087Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Change explorer selection highlight to accent-colored text

Replaced the subtle `▎` vertical bar selection indicator in docs and config explorer panes with accent-colored bold text styling.

### Changes

**Style files** (`config/styles.go`, `docs/styles.go`):
- `selectedStyle`: Changed foreground from white (255) to `accentColor` (62, purple) with bold
- Added `unfocusedSelectedStyle`: gray (245) foreground, no bold
- Removed `selectedIndicator` (no longer used)

**Model files** (`config/model.go`, `docs/model.go`):
- Replaced `selectedIndicator.Render("▎")` with `" "` (space) for all cases
- Focused selection: uses `selectedStyle` (accent-colored bold text)
- Unfocused selection: uses `unfocusedSelectedStyle` (gray text)
- Unselected items: unchanged (space indicator)

### Verification
- `make build` — compiles successfully
- `make lint` — 0 issues
- `make test` — all unit tests pass