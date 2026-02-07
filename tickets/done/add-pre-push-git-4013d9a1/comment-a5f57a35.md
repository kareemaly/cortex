---
id: a5f57a35-c0b8-4efc-b15d-61a9d59ac486
author: claude
type: review_requested
created: 2026-02-04T13:24:20.285754Z
action:
    type: git_diff
    args:
        commit: 407b13c
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/fe1d85f4-0a45-43cf-ba42-4e26b2b15ead
---
## Summary

Added a pre-push git hook that runs `make lint` and `make build` before allowing pushes.

## Changes

1. **`.githooks/pre-push`** (new file) - Bash script that:
   - Runs `make lint` first, exits with error if it fails
   - Runs `make build` second, exits with error if it fails
   - Displays colored output for success/failure messages
   - Can be bypassed with `git push --no-verify`

2. **`Makefile`** - Added `setup-hooks` target:
   - Configures `git config core.hooksPath .githooks`
   - Added to `.PHONY` list

3. **`CONTRIBUTING.md`** (new file) - Minimal documentation covering:
   - Development prerequisites and build commands
   - Git hooks setup instructions
   - Bypass instructions for emergencies

## Verification

- Ran `make setup-hooks` successfully
- Executed hook directly - both lint and build passed
- Git hooks path configured correctly