---
id: 0382e9a8-3810-4cfc-af60-11cd24271a1b
author: claude
type: review_requested
created: 2026-02-05T09:54:03.623136Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Research Summary: GitHub Copilot CLI Integration

### Key Discovery

There are **two distinct Copilot CLI tools**:

1. **`gh copilot`** (GitHub CLI Extension) - Simple helper for shell commands, limited use for Cortex
2. **`copilot`** (Standalone Agentic CLI) - Full-featured AI coding assistant, very similar to Claude Code

The standalone `copilot` CLI is the relevant one for Cortex integration. It's essentially GitHub's answer to Claude Code with:
- Multiple AI models (Claude Sonnet/Haiku/Opus, GPT-5.x, Gemini)
- MCP server support (compatible with Claude's format!)
- Session management with resume/continue
- Tool permissions system
- Non-interactive mode (`-p` flag)
- AGENTS.md for custom instructions

### Integration Feasibility: **HIGH**

Copilot CLI is architecturally similar to Claude Code, making integration straightforward.

### Recommended Implementation

**Phase 1: Config Changes**
```go
// internal/project/config/config.go
const AgentCopilot AgentType = "copilot"
```

**Phase 2: Launcher Script Generation**
Modify `internal/core/spawn/launcher.go` to support multiple agents:

```yaml
# .cortex/cortex.yaml example
architect:
  agent: copilot
  args: ["--model", "claude-sonnet-4.5", "--yolo"]

ticket:
  work:
    agent: claude  # or copilot
```

**Phase 3: Flag Mapping**

| Claude Flag | Copilot Equivalent | Notes |
|-------------|-------------------|-------|
| `--mcp-config` | `--additional-mcp-config` | JSON path |
| `--system-prompt` | N/A | Use AGENTS.md file |
| `--resume <id>` | `--resume <id>` | Same behavior |
| N/A | `--yolo` | Required for automation |
| N/A | `--model <name>` | Model selection |

**Phase 4: System Prompt Delivery**

Option A (Recommended): Generate temporary AGENTS.md
```
.cortex/tmp/AGENTS.md  # Write system prompt here
```
Then run copilot from that directory.

Option B: Use `--no-custom-instructions` and rely purely on MCP tools.

### Code Changes Required

1. `internal/project/config/config.go` - Add `AgentCopilot` type
2. `internal/core/spawn/launcher.go` - Add `buildCopilotLauncherScript()` 
3. `internal/core/spawn/spawn.go` - Route to correct launcher based on agent type
4. `.cortex/defaults/` - Add copilot-specific prompt templates

### Trade-offs

**Pros:**
- Multi-model flexibility (choose Claude, GPT, or Gemini)
- GitHub-native integration (PRs, issues, workflows)
- Plugin marketplace for extensions
- Compatible MCP format
- Autopilot mode for autonomous completion

**Cons:**
- Requires GitHub Copilot subscription
- System prompts delivered via files (less flexible than CLI flags)
- Different session ID management
- Tool permission model requires `--yolo` for full automation

### Prompting Best Practices

1. Use **AGENTS.md** in working directory for role-specific instructions
2. Keep instructions under 2 pages, focusing on:
   - Project overview and architecture
   - Build/test commands
   - Validation procedures
3. For non-interactive: `copilot -p "task" --yolo --additional-mcp-config <path>`
4. Grant file access with `--add-dir <project-path>`

### Recommendation

**Proceed with integration** as a secondary agent option. The architectural similarity makes it low-risk, and it provides valuable model flexibility for users who want to use GPT or different Claude versions through Copilot's multi-model support.

Estimated scope: ~200-300 lines of code changes across 4-5 files.