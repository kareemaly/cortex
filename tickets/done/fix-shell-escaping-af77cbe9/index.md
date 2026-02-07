---
id: af77cbe9-41e9-4d8a-8497-3f84fa0aafa4
title: Fix Shell Escaping for Spawn Session Prompt
type: ""
created: 2026-01-21T15:10:34Z
updated: 2026-01-21T15:10:34Z
---
The prompt passed to `claude` command is not properly escaped, causing shell interpretation of special characters.

## Bug

**File:** `internal/daemon/mcp/tools_architect.go:429`

**Current:**
```go
claudeCmd := fmt.Sprintf("claude %q --mcp-config %s --permission-mode plan", prompt, mcpConfigPath)
```

**Problem:** `%q` wraps the string in double quotes, but double quotes don't prevent:
- Backtick command substitution: `` `docs/` `` runs as a shell command
- Variable expansion: `$HOME` would expand
- History expansion: `!` could cause issues

**Observed error:**
```
zsh: no such file or directory: docs/
zsh: command not found: nYes
```

## Fix

Use single quotes and escape any single quotes in the prompt:

```go
// Escape single quotes for shell: ' → '\''
escapedPrompt := strings.ReplaceAll(prompt, "'", "'\\''")
claudeCmd := fmt.Sprintf("claude '%s' --mcp-config %s --permission-mode plan", escapedPrompt, mcpConfigPath)
```

Single quotes prevent all shell interpretation except for the quote character itself.

## Verification

```bash
# Create a ticket with backticks and special chars in body
# Spawn a session for it
# Verify the agent receives the full prompt without shell errors

make test
```

## Notes

- The `'\''` pattern works by: ending single quote, adding escaped single quote, starting new single quote
- This is the standard POSIX way to include single quotes in single-quoted strings

## Implementation

### Commits pushed
- `7024ac1` fix: use single quotes for spawn session prompt to prevent shell expansion

### Key files changed
- `internal/daemon/mcp/tools_architect.go` - Changed prompt escaping from `%q` (double quotes) to single quotes with POSIX escaping

### Important decisions
- Used `strings.ReplaceAll(prompt, "'", "'\\''")` for POSIX-compliant single quote escaping
- Added inline comments explaining the escaping approach

### Scope changes
- None - implementation matched original ticket specification