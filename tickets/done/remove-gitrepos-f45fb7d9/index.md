---
id: f45fb7d9-368e-4d98-aaa1-611b3f1641a6
title: Remove git.repos Config
type: ""
created: 2026-01-24T14:51:33Z
updated: 2026-01-24T14:51:33Z
---
## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Problem

`git.repos` config field exists but is never used. Dead code.

## Requirements

- Remove `GitConfig.Repos` and `RepoConfig` from config struct
- Remove validation for repos
- Update tests
- Simplify `GitConfig` to just hold future fields (e.g., `worktrees: bool`)

## Implementation

### Commits
- `f1f5612` refactor: remove unused git.repos config field

### Key Files Changed
- `internal/project/config/config.go` - Removed `RepoConfig` struct, `Repos` field from `GitConfig`, repos validation, and `itoa` helper
- `internal/project/config/config_test.go` - Removed repos-related test assertions and deleted `TestValidate_EmptyRepoPath`

### Decisions
- Kept `GitConfig` struct with just `Worktrees` field rather than flattening to `Config.Worktrees` to allow for future git-related config additions

### Scope Changes
- None, implemented as specified