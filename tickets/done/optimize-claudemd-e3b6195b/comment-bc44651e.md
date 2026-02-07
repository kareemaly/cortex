---
id: bc44651e-6f06-446e-84b9-cdc790c8fb11
author: claude
type: review_requested
created: 2026-02-04T14:24:11.023453Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Summary

Optimized CLAUDE.md based on best practices by adding 5 new sections while keeping the file concise (147 lines, up from 110).

### Changes Made

**New Sections:**
1. **Quick Start** - Essential 3-command workflow (`cortexd &`, `cortex init`, `cortex architect`)
2. **Critical Implementation Notes** - Key architectural constraints (HTTP-only communication, project context headers, StoreManager as source of truth, spawn state detection)
3. **Anti-Patterns** - Table showing common mistakes and correct alternatives (direct file access, direct tmux spawning, wrong imports)
4. **Debugging** - Troubleshooting table for common issues (daemon not responding, ticket not found, session won't spawn, MCP tools not working)

**Updated:**
- **CLI Commands table** - Added 3 missing commands from README.md: `cortex daemon status`, `cortex upgrade`, `cortex eject <path>`

### File Structure After Changes
```
# Cortex - AI Development Workflow
## Quick Start                    <- NEW
## Build & Test
## Architecture
## Critical Implementation Notes  <- NEW
## Anti-Patterns                  <- NEW
## Debugging                      <- NEW
## Key Paths
## Configuration
## CLI Commands                   <- UPDATED (+3 commands)
## API Endpoints
## MCP Tools
## Agent Workflow
## Lifecycle Hooks
## Testing
```

Final line count: 147 lines (target was 200-300, kept concise)