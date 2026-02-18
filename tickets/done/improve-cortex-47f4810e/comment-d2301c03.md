---
id: d2301c03-2c12-49b3-b369-1c037ff8b3cb
author: claude
type: review_requested
created: 2026-02-14T12:35:10.42652Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
        commit: a1673ae
---
## Changes

Single file modified: `cmd/cortex/commands/defaults_upgrade.go` (+51, -3 lines)

### 1. ANSI color constants
Added constants (`ansiReset`, `ansiRed`, `ansiGreen`, `ansiYellow`, `ansiCyan`, `ansiBold`) for colored terminal output. No new dependencies — uses raw escape codes.

### 2. Colored diff output
- `colorizeDiffLine()` — colors a single diff line based on its prefix (`---`/`+++` → bold, `@@` → cyan, `-` → red, `+` → green)
- `colorizeDiff()` — applies colorization to the full diff string
- `isColorEnabled()` — checks if stdout is a TTY using the existing `go-isatty` import
- Colorization is applied in `runDefaultsUpgradeApplyAll` where diffs are printed; piped/redirected output remains plain text

### 3. Prominent confirmation prompt
- `promptConfirmation()` now prints a blank line separator before the prompt
- When on a TTY, the prompt is rendered in bold yellow
- The prompt message now includes a `⚠` warning symbol: `⚠  Proceed with upgrade? [y/N]:`
- Default-no behavior is unchanged (empty input returns false)

### Verification
- `make build` — compiles successfully
- `make lint` — 0 issues
- `make test` — all unit tests pass