---
id: d19c4c42-fc64-461c-bdfe-35ba4b8ed3a7
author: claude
type: done
created: 2026-01-30T09:35:11.534367Z
---
## Summary

Implemented embedded default config folders using Go's `embed` package. Default configuration files now live in the source tree at `internal/install/defaults/` and are bundled into the binary, then copied to `~/.cortex/defaults/` on init.

## Key Changes

### New Files Created
- `internal/install/defaults/claude-code/cortex.yaml` - Default project configuration
- `internal/install/defaults/claude-code/prompts/architect/SYSTEM.md` - Architect system prompt
- `internal/install/defaults/claude-code/prompts/architect/KICKOFF.md` - Architect kickoff prompt  
- `internal/install/defaults/claude-code/prompts/ticket/work/SYSTEM.md` - Ticket system prompt
- `internal/install/defaults/claude-code/prompts/ticket/work/KICKOFF.md` - Ticket kickoff prompt
- `internal/install/defaults/claude-code/prompts/ticket/work/APPROVE.md` - Ticket approve prompt
- `internal/install/embed.go` - Contains `//go:embed defaults/*` directive and `copyEmbeddedDefaults()` function
- `internal/install/embed_test.go` - Tests for copy, no-overwrite, and force behaviors

### Modified Files
- `internal/install/install.go` - Removed ~150 lines of prompt constants and programmatic file generation, replaced `setupClaudeCodeDefaults()` with a 4-line function calling `copyEmbeddedDefaults()`

### Deleted Files
- `internal/install/prompts.go` - Content moved to embedded markdown files

## Key Decisions

1. **Used `//go:embed defaults/*`** - Single directive includes all subdirectories, making it easy to add new configs (just create `defaults/cursor/`, etc.)

2. **Created target directory first in `copyEmbeddedDir()`** - The `fs.WalkDir` skips the root "." entry, so we create the target directory before walking to ensure it exists

3. **Used `t.TempDir()` in tests** - More idiomatic than manual `os.MkdirTemp` + `defer os.RemoveAll()`, and satisfies the linter

4. **Preserved existing behavior** - Files are NOT overwritten unless `--force` is specified, preserving user customizations

## Verification

- All unit tests pass (`make test`)
- Linting passes (`make lint`)
- Build succeeds (`make build`)
- Manual testing confirmed:
  - `cortex init --global-only` creates all expected files
  - Existing files are preserved on subsequent runs
  - `--force` flag correctly overwrites files

## Commit
`7aebe4e` - feat(install): embed default configs and copy on init