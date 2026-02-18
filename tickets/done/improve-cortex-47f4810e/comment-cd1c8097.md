---
id: cd1c8097-a605-42fc-91c7-ec16f5e31308
author: claude
type: done
created: 2026-02-14T12:35:50.419893Z
---
## Summary

Improved the `cortex defaults upgrade` command UX with colored diff output and a prominent confirmation prompt.

### Changes

**File:** `cmd/cortex/commands/defaults_upgrade.go` (+51, -3 lines)

1. **ANSI color constants** — Added `ansiReset`, `ansiRed`, `ansiGreen`, `ansiYellow`, `ansiCyan`, `ansiBold` using raw escape codes (no new dependencies).

2. **Colored diff output** — Three new helpers:
   - `isColorEnabled()` — TTY detection via existing `go-isatty`
   - `colorizeDiffLine()` — red for removals, green for additions, cyan for hunk headers, bold for file headers
   - `colorizeDiff()` — applies colorization to full diff string; no-ops when piped/redirected

3. **Prominent confirmation prompt** — `promptConfirmation()` now adds a blank line separator, renders bold yellow on TTYs, and uses a `⚠` warning symbol.

### Verification
- `make build` ✓
- `make lint` — 0 issues ✓
- `make test` — all pass ✓
- Pre-push hooks pass ✓

### Commit
- `a1673ae` — pushed to `origin/main`