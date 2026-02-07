---
id: 5bfc6c00-76f1-4aad-acca-3a37044d333d
author: claude
type: review_requested
created: 2026-02-02T16:31:54.014938Z
action:
    type: git_diff
    args:
        commit: 69f58af
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/58a1e4c8-92e4-4f81-84e6-d65ab6f6b571
---
## Summary

Fixed the `DisplayPopup` function to ensure commands run from the correct working directory by wrapping the command with an explicit `cd`.

## Changes

**`internal/tmux/popup.go`**:
- Added `fmt` import
- When `workingDir` is specified, wrap the command with `cd %q && %s` to explicitly change to the directory before execution
- Kept the `-d` flag for initial directory context (belt and suspenders approach)

## Why

The tmux `-d` flag alone may not be sufficient for some tools like lazygit to properly respect the working directory. By explicitly wrapping the command with `cd <path> &&`, we guarantee the command runs from the correct repo directory in multi-repo projects.

## Verification

- ✅ Build passes (`make build`)
- ✅ Lint passes (`make lint`) - 0 issues
- ✅ All unit tests pass (`make test`)