---
id: a4102e27-198b-4fe4-a4a8-4163b909e98b
author: claude
type: review_requested
created: 2026-02-04T13:21:30.928858Z
action:
    type: git_diff
    args:
        commit: b6a8a5a
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/2c7dc98e-8fe2-4df3-ac0b-190ec9f05de7
---
## Summary

Added `cortex upgrade` command for self-updating cortex and cortexd binaries from GitHub releases.

## Files Created

**`internal/upgrade/` package:**
- `github.go` - GitHub API client for fetching release info
- `checksum.go` - Download and SHA256 verification
- `binary.go` - Binary detection, backup, replacement, code signing
- `upgrade.go` - Main orchestration logic

**`cmd/cortex/commands/upgrade.go`** - CLI command with flags:
- `--check` / `-c` - Only check if update is available
- `--version` / `-v` - Upgrade to specific version

## Features

1. Fetches latest (or specific) release from GitHub API
2. Downloads checksums.txt and verifies SHA256 of binaries
3. Backs up existing binaries to `~/.cortex/backups/`
4. Handles sudo for write-protected directories (e.g., /usr/local/bin)
5. Code signs on macOS
6. Stops daemon before upgrade, restarts after
7. Cleans up old backups (keeps last 3)
8. "dev" version always allows upgrade

## Verification

- Build passes: `make build`
- Lint passes: `make lint` (0 issues)
- Tests pass: `make test`
- Upgrade --check fails with 404 (expected - no release exists yet)
- Help output displays correctly