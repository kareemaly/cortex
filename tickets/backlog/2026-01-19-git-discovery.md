# Git Discovery

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
