---
id: e1deaa06-69f2-4209-b112-fa85fc8edc17
author: claude
type: review_requested
created: 2026-01-30T09:33:00.483976Z
action:
    type: git_diff
    args:
        commit: 7aebe4e
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Summary

Replaced programmatic default generation with Go's `embed` package. Default config files now live in the source tree and are bundled into the binary, then copied to `~/.cortex/defaults/` on init.

## Changes

### New Files
- `internal/install/defaults/claude-code/cortex.yaml` - Default project configuration
- `internal/install/defaults/claude-code/prompts/architect/SYSTEM.md` - Architect system prompt
- `internal/install/defaults/claude-code/prompts/architect/KICKOFF.md` - Architect kickoff prompt
- `internal/install/defaults/claude-code/prompts/ticket/work/SYSTEM.md` - Ticket system prompt
- `internal/install/defaults/claude-code/prompts/ticket/work/KICKOFF.md` - Ticket kickoff prompt
- `internal/install/defaults/claude-code/prompts/ticket/work/APPROVE.md` - Ticket approve prompt
- `internal/install/embed.go` - Embed directive and copy functions
- `internal/install/embed_test.go` - Tests for embed functionality

### Modified Files
- `internal/install/install.go` - Removed prompt constants, simplified `setupClaudeCodeDefaults` to use embedded files

### Deleted Files
- `internal/install/prompts.go` - Content moved to embedded files

## Testing

1. All unit tests pass (`make test`)
2. Linting passes (`make lint`)
3. Build succeeds (`make build`)
4. Manual verification:
   - Deleted `~/.cortex/defaults/claude-code/` and ran `cortex init --global-only` → correctly recreated
   - Modified a file and ran init again → file NOT overwritten (preserves user customizations)
   - Ran `cortex init --global-only --force` → file correctly overwritten

## Future Extensibility

To add a new default config (e.g., `cursor`):
1. Create `internal/install/defaults/cursor/` with config files
2. The `//go:embed defaults/*` directive automatically includes it
3. Call `copyEmbeddedDefaults("cursor", targetDir, force)`