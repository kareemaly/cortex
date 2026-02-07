---
id: 9608c3b3-8305-40f7-b45c-ec494f768590
title: Fix tmux send-keys Command Too Long
type: ""
created: 2026-01-26T10:05:26Z
updated: 2026-01-26T10:05:26Z
---
## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Problem

`cortex architect` fails with:

```
failed to spawn agent in tmux: tmux command failed: send-keys: command too long
```

The full claude command (prompt + system prompt + flags) is passed as a single argument to tmux `send-keys`. With many tickets, the prompt alone can exceed tmux's ~4KB buffer limit. POSIX shell escaping (single quote → `'\''`) makes it worse by roughly doubling size.

The chain:
1. `internal/core/spawn/spawn.go` builds architect prompt with full ticket list inline
2. `internal/core/spawn/command.go` embeds prompt + system prompt as inline arguments in the claude command string
3. `internal/tmux/command.go:20` passes the entire string to `send-keys` as one argument

The two largest contributors are:
- The ticket list prompt (scales with project size)
- The system prompt from `--append-system-prompt`

## Requirements

- Prompts and system prompts must not be passed inline in the tmux send-keys command
- Write prompt content to temporary files and use file-based arguments instead (e.g., `cat file | claude --append-system-prompt ...` or similar)
- Must work for both architect and ticket agent spawns
- Clean up temp files when session ends

## Implementation

### Commits

- `f3f9876` fix: replace inline prompt embedding with launcher script for tmux send-keys

### Key files changed

- **Created** `internal/core/spawn/prompt_file.go` — `WritePromptFile`/`RemovePromptFile` for writing prompt content to temp files
- **Created** `internal/core/spawn/launcher.go` — `WriteLauncherScript` generates a bash launcher that uses `$(cat file)` to read prompts at runtime; includes `trap EXIT` to clean up all temp files
- **Modified** `internal/core/spawn/command.go` — Removed `BuildClaudeCommand`, `ClaudeCommandParams`, `EscapePromptForShell` (replaced by launcher)
- **Modified** `internal/core/spawn/spawn.go` — `Spawn()` and `Resume()` now write prompt files, generate a launcher script, and pass `bash /path/launcher.sh` (~50 bytes) to tmux instead of the full inline command; `cleanupOnFailure()` changed to accept `tempFiles []string`
- **Modified** `internal/core/spawn/spawn_test.go` — Replaced old command-building tests with launcher script tests; updated spawn/resume integration tests to verify launcher usage

### Decisions

- **`$(cat file)` in double quotes** is safe — parameter expansion is not re-interpreted by bash, so prompt content with backticks, `$`, or quotes is passed verbatim to claude
- **`trap EXIT`** handles cleanup for all exit paths — normal exit, error, SIGHUP from tmux kill-window. This also cleans up MCP and settings config files, which were previously leaked on normal session completion
- **Removed dead code** (`BuildClaudeCommand`, `EscapePromptForShell`) rather than keeping backward compatibility — per ticket guidance on no tech debt

### Scope changes

None — implemented as specified