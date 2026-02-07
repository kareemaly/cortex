---
id: b5fa510f-5d10-4f0b-8b82-fb7f5dc1d712
title: Fix claude CLI argument order when spawning sessions
type: ""
created: 2026-01-21T11:47:02Z
updated: 2026-01-21T11:47:02Z
---
When spawning architect or ticket sessions, the `--mcp-config` flag is placed before the prompt, causing Claude to hang.

## Problem

Current command generation:
```bash
claude --mcp-config /tmp/config.json "prompt"  # HANGS
```

Correct order:
```bash
claude "prompt" --mcp-config /tmp/config.json  # WORKS
```

The prompt must come immediately after `claude`, with flags after.

## Files to fix

Search for where claude commands are constructed:
- `cmd/cortex/commands/architect.go` - architect session spawning
- `internal/daemon/mcp/tools_architect.go` - ticket session spawning via MCP
- `internal/tmux/command.go` - if command building happens here

## Requirements

Reorder arguments so prompt comes first, then `--mcp-config` flag.

## Verification

```bash
make build && cp bin/cortex bin/cortexd ~/.local/bin/

cd ~/projects/test-cortex
cortex architect  # Should not hang
```

## Implementation

### Commits pushed
- `c9be14f` fix: place prompt before flags in claude CLI commands to prevent hanging

### Key files changed
- `cmd/cortex/commands/architect.go` - Fixed `buildAgentCommand()` to place prompt before `--mcp-config` for both opencode and claude agents
- `internal/daemon/mcp/tools_architect.go` - Fixed `handleSpawnSession()` to place prompt before flags
- `internal/daemon/api/tickets.go` - Fixed `buildAgentCommand()` to place `-p 'prompt'` before `--mcp-config`

### Important decisions
- Applied consistent ordering across all three locations where claude/opencode commands are constructed
- Prompt always comes immediately after the command name, flags follow after

### Scope changes
- None - implemented as specified in ticket