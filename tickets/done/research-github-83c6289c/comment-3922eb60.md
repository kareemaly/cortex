---
id: 3922eb60-5bd1-4292-85c4-a5fe5b49fc0a
author: claude
type: comment
created: 2026-02-05T09:50:21.418443Z
---
## Integration Analysis: Cortex + Copilot CLI

### Current Cortex Agent Architecture
- Agents defined in `.cortex/cortex.yaml` as `AgentType` (currently: `claude`, `opencode`)
- `RoleConfig` has `agent` type and `args` (extra CLI args)
- Launcher script (`launcher.go`) currently hardcoded to call `claude`
- MCP config uses Claude's `mcpServers` format
- System/kickoff prompts loaded from `.cortex/prompts/{role}/{type}/`

### Key Integration Points

1. **Config Changes** (`internal/project/config/config.go`)
   - Add `AgentCopilot AgentType = "copilot"` constant
   - Update `Validate()` to accept `copilot` type

2. **Launcher Script** (`internal/core/spawn/launcher.go`)
   - Accept agent type parameter
   - Build `copilot` command instead of `claude` when configured
   - Map Claude flags to Copilot equivalents

3. **Flag Mapping** (Claude → Copilot):
   - `--mcp-config` → `--additional-mcp-config`
   - `--system-prompt` → No direct equivalent (uses AGENTS.md)
   - `--append-system-prompt` → No direct equivalent
   - `--resume <session>` → `--resume <session>`
   - `--session-id` → No equivalent (Copilot auto-manages)

4. **MCP Config Format**
   - Both use similar JSON format with `mcpServers` key
   - Copilot also supports Claude-style `.mcp.json` (per changelog)
   - Should be compatible without changes

### Challenges

1. **System Prompts**: Copilot uses AGENTS.md files, not CLI flags
   - Would need to generate temporary AGENTS.md in working directory
   - Or use `--no-custom-instructions` and rely on MCP tools only

2. **Tool Permissions**: Copilot requires explicit permission grants
   - Add `--yolo` or `--allow-all-tools` for automation
   - Or grant specific permissions via `--allow-tool`

3. **MCP Tool Binding**: Both support MCP, so Cortex's MCP tools should work
   - Need to test compatibility with Copilot's MCP implementation