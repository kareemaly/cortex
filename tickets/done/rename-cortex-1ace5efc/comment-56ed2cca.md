---
id: 56ed2cca-7e98-47b0-94f8-fea31edc530c
author: claude
type: ticket_done
created: 2026-01-27T11:05:29.689077Z
---
## Summary

Renamed the `cortex install` CLI command to `cortex init` to follow the universal CLI convention used by tools like `git init`, `npm init`, etc.

## Changes Made

### Files Modified
1. **`cmd/cortex/commands/init.go`** (new, replaces `install.go`)
   - Command `Use` changed from `"install"` to `"init"`
   - `Short` description updated to `"Initialize project"`
   - `Long` description updated accordingly
   - Variables renamed: `installCmd` → `initCmd`, `installGlobalOnly` → `initGlobalOnly`, `installForce` → `initForce`
   - Function renamed: `runInstall` → `runInit`
   - Helper functions (`printItems`, `checkMark`, `crossMark`, `bullet`) moved here from deleted file

2. **`cmd/cortex/commands/install.go`** (deleted)

3. **`internal/prompt/errors.go`** (line 11)
   - Error message changed from `"cortex install"` to `"cortex init"`

4. **`internal/prompt/prompt_test.go`** (line 66)
   - Test assertion updated from `"cortex install"` to `"cortex init"`

5. **`DESIGN.md`** (line 52)
   - CLI commands listing updated from `cortex install` to `cortex init`

## Key Decisions
- **`internal/install/` package left unchanged**: Go does not allow `init` as a package name since `init()` is a special function. The package is an internal implementation detail with no user-facing exposure.
- **Only renamed user-facing references**: `CLAUDE.md`, `Makefile`, and prompt templates had no references to the install command and were not modified.

## Verification
- `make build` — compiles successfully
- `make test` — all unit tests pass (including updated prompt test)
- `bin/cortex init --help` — new command works correctly
- `bin/cortex install` — correctly returns "unknown command"

## Commit
`cd4a4fb` — rename `cortex install` to `cortex init` to follow standard CLI conventions