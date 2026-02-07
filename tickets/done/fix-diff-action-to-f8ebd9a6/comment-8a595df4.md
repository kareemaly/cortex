---
id: 8a595df4-37cb-4791-b9e0-8723ce5dcedf
author: claude
type: done
created: 2026-02-02T16:40:26.944246Z
---
## Summary

Fixed the `DisplayPopup` function in `internal/tmux/popup.go` to ensure commands run from the correct working directory in multi-repo projects.

## Problem

The tmux `-d` flag alone was insufficient for some tools (like lazygit) to properly respect the working directory when launched via popup.

## Solution

Modified `DisplayPopup` to wrap the command with `cd <workingDir> && <command>` when a working directory is specified. This guarantees the command explicitly runs from the correct repo directory.

## Changes

- **File**: `internal/tmux/popup.go`
- Added `fmt` import
- When `workingDir` is set, command is wrapped: `fmt.Sprintf("cd %q && %s", workingDir, command)`
- Kept the `-d` flag for initial directory context (belt and suspenders approach)

## Verification

- Build passes
- Lint passes (0 issues)
- All unit tests pass

## Commit

`69f58af` - fix(tmux): wrap popup command with cd to ensure correct working directory