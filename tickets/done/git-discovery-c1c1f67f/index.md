---
id: c1c1f67f-840f-49ca-bb45-b18872adf00c
title: Git Discovery
type: ""
created: 2026-01-20T13:07:38Z
updated: 2026-01-20T13:07:38Z
---
Implement git repository discovery and status tracking for tickets.

## Context

Tickets track git state via `git_base` (commit SHA when session started) and sessions report file changes. Projects can have multiple repos (monorepo support).

See `DESIGN.md` for:
- Git base tracking in session schema (lines 102-104, 121-123)
- Project config git.repos (lines 279-282)
- Session view showing git diff stats (lines 339-342)

## Requirements

Create `internal/git/` package that:

1. **Repository Discovery**
   - Find .git directories from project config paths
   - Support multiple repos per project
   - Handle nested repos (monorepo)

2. **State Queries**
   - Get current branch name
   - Get current commit SHA (short and full)
   - Check if repo is clean (no uncommitted changes)

3. **Diff Operations**
   - Get diff stats since a base commit (files changed, insertions, deletions)
   - List changed files since base commit
   - Support comparing across branches

4. **Utilities**
   - Validate path contains git repo
   - Get repo root from any subdirectory

## Verification

```bash
make build   # Builds successfully
make test    # Tests pass
make lint    # No lint errors
```

## Notes

- Use `os/exec` to shell out to git commands
- Handle cases where git isn't installed gracefully
- Parse git output carefully (porcelain formats where available)
- Tests can use temporary git repos created in test setup

## Implementation

### Commits

- `4dde860` feat: add git discovery package for repository detection and status
- `069c29e` Merge branch 'ticket/2026-01-19-git-discovery'

### Key Files

| File | Description |
|------|-------------|
| `internal/git/errors.go` | Custom error types with helper functions |
| `internal/git/git.go` | Core `Repo` type, discovery functions, git command helpers |
| `internal/git/status.go` | State queries (branch, commit, clean status) |
| `internal/git/diff.go` | Diff stats and changed file operations |
| `internal/git/git_test.go` | Unit tests for all public functions |
| `internal/git/testutil_test.go` | Test helpers for creating temp git repos |

### Decisions

- Used `exec.Command` to shell out to git rather than a Go git library for simplicity and reliability
- Added `DiscoverRepos` with deduplication to support projects with multiple repos
- Created typed errors (`GitNotInstalledError`, `NotARepoError`, `CommitNotFoundError`) with helper functions (`IsNotARepo()`, etc.) following existing codebase patterns
- Tests use temporary git repos with full commit history for realistic testing

### Scope

Implemented as specified. No scope changes from original ticket.