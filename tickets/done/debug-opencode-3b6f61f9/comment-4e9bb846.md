---
id: 4e9bb846-90ae-4f30-9425-75848d2c6c7d
author: claude
type: comment
created: 2026-02-12T11:38:32.305002Z
---
## Root Cause Analysis

**The bug**: OpenCode architect sessions print help/usage text and exit immediately instead of starting an interactive session.

**Root cause**: The opencode defaults config (`internal/install/defaults/opencode/cortex.yaml`) contains Claude Code-specific CLI args that OpenCode does not recognize.

### Trace

1. The opencode defaults file defines args for every role that are copied verbatim from the claude-code defaults:
   ```yaml
   architect:
     agent: opencode
     args:
       - "--allow-dangerously-skip-permissions"  # Claude Code flag
       - "--allowedTools"                          # Claude Code flag
       - "mcp__cortex__listTickets,mcp__cortex__readTicket"
   ```

2. These args flow through: `projectconfig.Load()` → `architect.go:spawnArchitectSession()` → `SpawnRequest.AgentArgs` → `LauncherParams.AgentArgs` → `buildOpenCodeCommand()` (launcher.go:167-169)

3. The resulting command becomes:
   ```
   opencode --agent cortex --prompt "$(cat ...)" '--allow-dangerously-skip-permissions' '--allowedTools' 'mcp__cortex__listTickets,...'
   ```

4. OpenCode doesn't recognize `--allow-dangerously-skip-permissions` or `--allowedTools` → prints help → exits.

### Why the args are redundant for OpenCode

Permissions for OpenCode are already configured via `OPENCODE_CONFIG_CONTENT` env var in `opencode_config.go:39`:
```go
Permission: map[string]string{"*": "allow"}
```

This replaces both `--allow-dangerously-skip-permissions` and `--allowedTools`. The CLI args are not needed.

### Scope

This affects ALL opencode session types (architect, meta, ticket), not just architect. The opencode defaults have invalid Claude-specific args for every role.