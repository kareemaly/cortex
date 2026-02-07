---
id: 8d652fa4-d114-4b10-9f64-c9c5a6285c46
author: claude
type: review_requested
created: 2026-02-05T09:16:52.012484Z
action:
    type: git_diff
    args:
        commit: 13a7c90
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Changes

### New CLI Command: `cortex defaults upgrade`

Refreshes `~/.cortex/defaults/` with the latest embedded defaults from the binary.

**Usage:**
- `cortex defaults upgrade` - Updates all default files (force overwrites)
- `cortex defaults upgrade --dry-run` - Preview changes without applying them

### Files Modified

1. **`internal/install/embed.go`**
   - Exported `copyEmbeddedDefaults` → `CopyEmbeddedDefaults`
   - Added `CompareStatus` type with `CompareUnchanged`, `CompareWillUpdate`, `CompareWillCreate`
   - Added `CompareItem` struct for comparison results
   - Added `CompareEmbeddedDefaults()` function for dry-run comparison

2. **`internal/install/install.go`**
   - Updated call to use exported `CopyEmbeddedDefaults`

3. **`internal/install/embed_test.go`**
   - Updated test function calls to use exported `CopyEmbeddedDefaults`

4. **`cmd/cortex/commands/defaults.go`** (NEW)
   - Parent command group following `ticket.go` pattern

5. **`cmd/cortex/commands/defaults_upgrade.go`** (NEW)
   - Upgrade subcommand with `--dry-run` flag
   - Formatted output with checkmarks, bullets, and summary

### Verification

- `make build` - ✓ Compiles
- `make lint` - ✓ No issues
- `make test` - ✓ All tests pass
- `cortex defaults upgrade --dry-run` - ✓ Shows preview correctly
- `cortex defaults upgrade` - ✓ Updates files correctly